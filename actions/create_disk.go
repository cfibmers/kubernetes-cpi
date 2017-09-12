package actions

import (
	"fmt"
	"reflect"
	"time"

	"code.cloudfoundry.org/clock"

	"github.ibm.com/Bluemix/kubernetes-cpi/cpi"
	"github.ibm.com/Bluemix/kubernetes-cpi/kubecluster"
	core "k8s.io/client-go/kubernetes/typed/core/v1"
	"k8s.io/client-go/pkg/api/resource"
	"k8s.io/client-go/pkg/api/v1"
	"k8s.io/client-go/pkg/labels"
	"k8s.io/client-go/pkg/watch"

	bosherr "github.com/cloudfoundry/bosh-utils/errors"
)

type CreateDiskCloudProperties struct {
	Context            string `json:"context"`
	StorageClass       string `json:"storage_class"`
	StorageProvisioner string `json:"storage_provisioner"`
}

// DiskCreator simply creates a PersistentVolumeClaim.
// True? --> The attach process will
// turn the claim into a volume mounted into the pod.
type DiskCreator struct {
	ClientProvider    kubecluster.ClientProvider
	Clock             clock.Clock
	DiskReadyTimeout  time.Duration
	GUIDGeneratorFunc func() (string, error)
}

func (d *DiskCreator) CreateDisk(size uint, cloudProps CreateDiskCloudProperties, vmcid cpi.VMCID) (cpi.DiskCID, error) {
	diskID, err := d.GUIDGeneratorFunc()
	if err != nil {
		return "", bosherr.WrapError(err, "Creating disk")
	}

	volumeSize, err := resource.ParseQuantity(fmt.Sprintf("%dGi", size))
	if err != nil {
		return "", bosherr.WrapError(err, "Parsing quantity")
	}

	client, err := d.ClientProvider.New(cloudProps.Context)
	if err != nil {
		return "", bosherr.WrapError(err, "Creating client")
	}

	// volumeName := "volume-" + diskID

	// _, err = client.PersistentVolumes().Create(&v1.PersistentVolume{
	// 	ObjectMeta: v1.ObjectMeta{
	// 		Name: volumeName,
	// 		Labels: map[string]string{
	// 			"bosh.cloudfoundry.org/disk-id": diskID,
	// 		},
	// 	},
	// 	Spec: v1.PersistentVolumeSpec{
	// 		AccessModes: []v1.PersistentVolumeAccessMode{v1.ReadWriteOnce},
	// 		Capacity: v1.ResourceList{
	// 			v1.ResourceStorage: volumeSize,
	// 		},
	// 		PersistentVolumeReclaimPolicy: v1.PersistentVolumeReclaimRecycle,
	// 	},
	// })

	// if err != nil {
	// 	return "", err
	// }

	pvc, err := client.PersistentVolumeClaims().Create(&v1.PersistentVolumeClaim{
		ObjectMeta: v1.ObjectMeta{
			Name:      "disk-" + diskID,
			Namespace: client.Namespace(),
			Annotations: map[string]string{
				"volume.beta.kubernetes.io/storage-class":       cloudProps.StorageClass,
				"volume.beta.kubernetes.io/storage-provisioner": cloudProps.StorageProvisioner,
			},
			Labels: map[string]string{
				"bosh.cloudfoundry.org/disk-id": diskID,
			},
		},
		Spec: v1.PersistentVolumeClaimSpec{
			// VolumeName:  volumeName,
			AccessModes: []v1.PersistentVolumeAccessMode{v1.ReadWriteMany},
			Resources: v1.ResourceRequirements{
				Requests: v1.ResourceList{
					v1.ResourceStorage: volumeSize,
				},
			},
		},
	})

	if err != nil {
		return "", bosherr.WrapError(err, "Creating PVC")
	}

	ready, err := d.waitForDisk(client.PersistentVolumeClaims(), diskID, pvc.ResourceVersion)

	if err != nil {
		return "", bosherr.WrapError(err, "Waiting for disk")
	}

	if !ready {
		return "", bosherr.Error("Disk creation failed with a timeout")
	}
	return NewDiskCID(client.Context(), diskID), nil
}

func (d *DiskCreator) waitForDisk(pvcService core.PersistentVolumeClaimInterface, diskID string, resourceVersion string) (bool, error) {
	diskSelector, err := labels.Parse("bosh.cloudfoundry.org/disk-id=" + diskID)
	if err != nil {
		return false, bosherr.WrapError(err, "Parsing disk selector")
	}

	listOptions := v1.ListOptions{
		LabelSelector:   diskSelector.String(),
		ResourceVersion: resourceVersion,
		Watch:           true,
	}

	timer := d.Clock.NewTimer(d.DiskReadyTimeout)
	defer timer.Stop()

	pvcWatch, err := pvcService.Watch(listOptions)
	if err != nil {
		return false, bosherr.WrapError(err, "Watching PVC")
	}
	defer pvcWatch.Stop()

	for {
		select {
		case event := <-pvcWatch.ResultChan():
			switch event.Type {
			case watch.Modified:
				pvc, ok := event.Object.(*v1.PersistentVolumeClaim)
				if !ok {
					return false, bosherr.Errorf("Unexpected object type: %v", reflect.TypeOf(event.Object))
				}

				if isDiskReady(pvc) {
					return true, nil
				}

			default:
				return false, bosherr.Errorf("Unexpected pvc watch event: %s", event.Type)
			}

		case <-timer.C():
			return false, nil
		}
	}
}

func isDiskReady(pvc *v1.PersistentVolumeClaim) bool {
	if pvc.Status.Phase != v1.ClaimBound {
		return false
	}

	return true
}
