package actions

import (
	"strings"

	"github.ibm.com/Bluemix/kubernetes-cpi/cpi"
	"github.ibm.com/Bluemix/kubernetes-cpi/kubecluster"
	"k8s.io/client-go/pkg/util/validation"

	bosherr "github.com/cloudfoundry/bosh-utils/errors"
)

type DiskMetadataSetter struct {
	ClientProvider kubecluster.ClientProvider
}

type Metadata struct {
	Labels      map[string]string
	Annotations map[string]string
}

func (v *DiskMetadataSetter) SetDiskMetadata(diskCID cpi.DiskCID, metadata map[string]string) error {
	context, diskID := ParseDiskCID(diskCID)

	client, err := v.ClientProvider.New(context)
	if err != nil {
		return bosherr.WrapError(err, "Creating client")
	}

	disk, err := client.PersistentVolumeClaims().Get("disk-" + diskID)
	if err != nil {
		return bosherr.WrapError(err, "Getting PVC")
	}

	if disk.ObjectMeta.Labels == nil {
		disk.ObjectMeta.Labels = map[string]string{}
	}

	for k, v := range metadata {
		if k == "attached_at" {
			v = strings.Replace(v, ":", "_", -1)
		}

		k = "bosh.cloudfoundry.org/" + k
		errs := validation.IsQualifiedName(k)
		if len(errs) > 0 {
			return bosherr.Errorf("Error setting disk metadata: \"%s\": \"%s\": %s", k, v, strings.Join(errs, ": "))
		}

		errs = validation.IsValidLabelValue(v)
		if len(errs) > 0 {
			return bosherr.Errorf("Error setting disk metadata: \"%s\": \"%s\": %s", k, v, strings.Join(errs, ": "))
		}

		disk.ObjectMeta.Labels[k] = v
	}

	_, err = client.PersistentVolumeClaims().Update(disk)
	if err != nil {
		return bosherr.WrapError(err, "Updating PVC")
	}

	return nil
}
