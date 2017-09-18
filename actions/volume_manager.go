package actions

import (
	"encoding/json"
	"reflect"
	"time"

	"code.cloudfoundry.org/clock"

	"github.ibm.com/Bluemix/kubernetes-cpi/agent"
	"github.ibm.com/Bluemix/kubernetes-cpi/cpi"
	"github.ibm.com/Bluemix/kubernetes-cpi/kubecluster"
	"k8s.io/client-go/pkg/api/v1"
	"k8s.io/client-go/pkg/labels"
	"k8s.io/client-go/pkg/watch"

	bosherr "github.com/cloudfoundry/bosh-utils/errors"
	core "k8s.io/client-go/kubernetes/typed/core/v1"
)

type VolumeManager struct {
	ClientProvider kubecluster.ClientProvider

	Clock             clock.Clock
	PodReadyTimeout   time.Duration
	PostRecreateDelay time.Duration
}

type Operation int

const (
	Add Operation = iota
	Remove
)

func (v *VolumeManager) AttachDisk(vmcid cpi.VMCID, diskCID cpi.DiskCID) error {
	vmContext, agentID := ParseVMCID(vmcid)
	context, diskID := ParseDiskCID(diskCID)
	if context != vmContext {
		return bosherr.Errorf("Kubernetes disk and resource pool contexts must be the same: disk: %q, resource pool: %q", context, vmContext)
	}

	client, err := v.ClientProvider.New(context)
	if err != nil {
		return bosherr.WrapError(err, "Creating client")
	}

	err = v.recreatePod(client, Add, agentID, diskID)
	if err != nil {
		return bosherr.WrapError(err, "Recreating pod to attach disk")
	}

	return nil
}

func (v *VolumeManager) DetachDisk(vmcid cpi.VMCID, diskCID cpi.DiskCID) error {
	vmContext, agentID := ParseVMCID(vmcid)
	context, diskID := ParseDiskCID(diskCID)
	if context != vmContext {
		return bosherr.Errorf("Kubernetes disk and resource pool contexts must be the same: disk: %q, resource pool: %q", context, vmContext)
	}

	client, err := v.ClientProvider.New(context)
	if err != nil {
		return bosherr.WrapError(err, "Creating client")
	}

	err = v.recreatePod(client, Remove, agentID, diskID)
	if err != nil {
		return bosherr.WrapError(err, "Recreating pod to detach disk")
	}

	return nil
}

func (v *VolumeManager) recreatePod(client kubecluster.Client, op Operation, agentID string, diskID string) error {
	podService := client.Pods()
	pod, err := podService.Get("agent-" + agentID)
	if err != nil {
		return bosherr.WrapError(err, "Getting pod")
	}

	err = updateConfigMapDisks(client, op, agentID, diskID)
	if err != nil {
		return bosherr.WrapError(err, "Updating disk configMap")
	}

	updateVolumes(op, &pod.Spec, diskID)

	if pod.Annotations == nil {
		pod.Annotations = map[string]string{}
	}

	if len(pod.Annotations["bosh.cloudfoundry.org/ip-address"]) == 0 {
		pod.Annotations["bosh.cloudfoundry.org/ip-address"] = pod.Status.PodIP
	}

	pod.ObjectMeta = v1.ObjectMeta{
		Name:        pod.Name,
		Namespace:   pod.Namespace,
		Annotations: pod.Annotations,
		Labels:      pod.Labels,
	}
	pod.Status = v1.PodStatus{}

	err = podService.Delete("agent-"+agentID, &v1.DeleteOptions{GracePeriodSeconds: int64Ptr(0)})
	if err != nil {
		return bosherr.WrapError(err, "Deleting pod")
	}

	updated, err := podService.Create(pod)
	if err != nil {
		return bosherr.WrapError(err, "Recreating pod")
	}

	if err := v.waitForPod(podService, agentID, updated.ResourceVersion); err != nil {
		return bosherr.WrapError(err, "Waiting for pod recreate")
	}

	// TODO: Need an agent readiness check that's real
	v.Clock.Sleep(v.PostRecreateDelay)

	return nil
}

func updateConfigMapDisks(client kubecluster.Client, op Operation, agentID, diskID string) error {
	configMapService := client.ConfigMaps()
	cm, err := configMapService.Get("agent-" + agentID)
	if err != nil {
		return bosherr.WrapError(err, "Getting configMaps")
	}

	var settings agent.Settings
	err = json.Unmarshal([]byte(cm.Data["instance_settings"]), &settings)
	if err != nil {
		return bosherr.WrapError(err, "Unmarshalling instance settings")
	}

	diskCID := string(NewDiskCID(client.Context(), diskID))
	if settings.Disks.Persistent == nil {
		settings.Disks.Persistent = map[string]string{}
	}

	switch op {
	case Add:
		settings.Disks.Persistent[diskCID] = "/mnt/" + diskID
	case Remove:
		delete(settings.Disks.Persistent, diskCID)
	}

	settingsJSON, err := json.Marshal(settings)
	if err != nil {
		return bosherr.WrapError(err, "instance settings")
	}

	cm.Data["instance_settings"] = string(settingsJSON)

	_, err = configMapService.Update(cm)
	if err != nil {
		return bosherr.WrapError(err, "Updating configMap")
	}

	return nil
}

func updateVolumes(op Operation, spec *v1.PodSpec, diskID string) {
	switch op {
	case Add:
		addVolume(spec, diskID)
	case Remove:
		removeVolume(spec, diskID)
	}
}

func addVolume(spec *v1.PodSpec, diskID string) {
	spec.Volumes = append(spec.Volumes, v1.Volume{
		Name: "disk-" + diskID,
		VolumeSource: v1.VolumeSource{
			PersistentVolumeClaim: &v1.PersistentVolumeClaimVolumeSource{
				ClaimName: "disk-" + diskID,
			},
		},
	})

	for i, c := range spec.Containers {
		if c.Name == "bosh-job" {
			spec.Containers[i].VolumeMounts = append(c.VolumeMounts, v1.VolumeMount{
				Name:      "disk-" + diskID,
				MountPath: "/mnt/" + diskID,
			})
			break
		}
	}
}

func removeVolume(spec *v1.PodSpec, diskID string) {
	for i, v := range spec.Volumes {
		if v.Name == "disk-"+diskID {
			spec.Volumes = append(spec.Volumes[:i], spec.Volumes[i+1:]...)
			break
		}
	}

	for i, c := range spec.Containers {
		if c.Name == "bosh-job" {
			for j, v := range c.VolumeMounts {
				if v.Name == "disk-"+diskID {
					spec.Containers[i].VolumeMounts = append(c.VolumeMounts[:j], c.VolumeMounts[j+1:]...)
					break
				}
			}
		}
	}
}

func (v *VolumeManager) waitForPod(podService core.PodInterface, agentID string, resourceVersion string) error {
	agentSelector, err := labels.Parse("bosh.cloudfoundry.org/agent-id=" + agentID)
	if err != nil {
		return bosherr.WrapError(err, "Parsing agent selector")
	}

	listOptions := v1.ListOptions{
		LabelSelector:   agentSelector.String(),
		ResourceVersion: resourceVersion,
		Watch:           true,
	}

	timer := v.Clock.NewTimer(v.PodReadyTimeout)
	defer timer.Stop()

	podWatch, err := podService.Watch(listOptions)
	if err != nil {
		return bosherr.WrapError(err, "Watching pod")
	}
	defer podWatch.Stop()

	for {
		select {
		case event := <-podWatch.ResultChan():
			switch event.Type {
			case watch.Modified:
				pod, ok := event.Object.(*v1.Pod)
				if !ok {
					return bosherr.Errorf("Unexpected object type: %v", reflect.TypeOf(event.Object))
				}

				if isAgentContainerRunning(pod) {
					return nil
				}

			default:
				return bosherr.Errorf("Unexpected pod watch event: %s", event.Type)
			}

		case <-timer.C():
			return bosherr.Error("Pod create failed with a timeout")
		}
	}
}

func isAgentContainerRunning(pod *v1.Pod) bool {
	if pod.Status.Phase != v1.PodRunning {
		return false
	}

	for _, containerStatus := range pod.Status.ContainerStatuses {
		if containerStatus.Name == "bosh-job" {
			return containerStatus.Ready && containerStatus.State.Running != nil
		}
	}

	return false
}
