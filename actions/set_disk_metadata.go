package actions

import (
	"strings"
	"encoding/json"

	"github.ibm.com/Bluemix/kubernetes-cpi/cpi"
	"github.ibm.com/Bluemix/kubernetes-cpi/kubecluster"
	"k8s.io/client-go/pkg/api"
	"k8s.io/client-go/pkg/util/strategicpatch"
	"k8s.io/client-go/pkg/util/validation"
)

type DiskMetadataSetter struct {
	ClientProvider kubecluster.ClientProvider
}

func (v *DiskMetadataSetter) SetDiskMetadata(diskCID cpi.DiskCID, metadata map[string]string) error {
	context, diskID := ParseDiskCID(diskCID)

	client, err := v.ClientProvider.New(context)
	if err != nil {
		return err
	}

	disk, err := client.PersistentVolumeClaims().Get("disk-" + diskID)
	if err != nil {
		return err
	}

	old, err := json.Marshal(disk)
	if err != nil {
		return err
	}

	for k, v := range metadata {
		if k == "attached_at" {
			v = strings.Replace(v, ":", "_", -1)
		}

		k = "bosh.cloudfoundry.org/" + strings.ToLower(k)
		if len(validation.IsQualifiedName(k)) == 0 && len(validation.IsValidLabelValue(v)) == 0 {
			disk.ObjectMeta.Labels[k] = v
		}
	}

	new, err := json.Marshal(disk)
	if err != nil {
		return err
	}

	patch, err := strategicpatch.CreateTwoWayMergePatch(old, new, disk)
	if err != nil {
		return err
	}

	_, err = client.PersistentVolumeClaims().Patch(disk.Name, api.StrategicMergePatchType, patch)
	if err != nil {
		return err
	}

	return nil
}