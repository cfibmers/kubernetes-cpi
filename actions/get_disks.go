package actions

import (
	"net/http"

	"github.ibm.com/Bluemix/kubernetes-cpi/cpi"
	"github.ibm.com/Bluemix/kubernetes-cpi/kubecluster"
	"k8s.io/client-go/pkg/api/errors"
	"k8s.io/client-go/pkg/api/v1"

	bosherr "github.com/cloudfoundry/bosh-utils/errors"
	core "k8s.io/client-go/kubernetes/typed/core/v1"
)

type DiskGetter struct {
	ClientProvider kubecluster.ClientProvider
}

func (d *DiskGetter) GetDisks(vmcid cpi.VMCID) ([]cpi.DiskCID, error) {
	context, agentID := ParseVMCID(vmcid)
	client, err := d.ClientProvider.New(context)
	if err != nil {
		return nil, bosherr.WrapError(err, "Creating client")
	}

	pod, err := client.Pods().Get("agent-" + agentID)
	if err != nil {
		if statusError, ok := err.(*errors.StatusError); ok {
			if statusError.Status().Code == http.StatusNotFound {
				return []cpi.DiskCID{}, nil
			}
		}
		return nil, bosherr.WrapError(err, "Getting pod")
	}

	diskIDs := []cpi.DiskCID{}
	for _, v := range pod.Spec.Volumes {
		pvc, err := getPVClaim(client.PersistentVolumeClaims(), v.VolumeSource)
		if err != nil && !isNotFoundStatusError(err) {
			return nil, bosherr.WrapError(err, "Getting PVC")
		}

		if pvc == nil {
			continue
		}

		if diskID, ok := pvc.Labels["bosh.cloudfoundry.org/disk-id"]; ok {
			diskIDs = append(diskIDs, NewDiskCID(context, diskID))
		}
	}

	return diskIDs, nil
}

func getPVClaim(pvcClient core.PersistentVolumeClaimInterface, volumeSource v1.VolumeSource) (*v1.PersistentVolumeClaim, error) {
	if volumeSource.PersistentVolumeClaim != nil {
		return pvcClient.Get(volumeSource.PersistentVolumeClaim.ClaimName)
	}
	return nil, nil
}

func isNotFoundStatusError(err error) bool {
	if statusErr, ok := err.(*errors.StatusError); ok {
		return statusErr.Status().Code == http.StatusNotFound
	}
	return false
}
