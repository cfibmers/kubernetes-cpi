package actions

import (
	"fmt"
	"strings"

	"github.ibm.com/Bluemix/kubernetes-cpi/cpi"
	"github.ibm.com/Bluemix/kubernetes-cpi/kubecluster"
	"k8s.io/client-go/pkg/util/validation"
)

type DiskMetadataSetter struct {
	ClientProvider kubecluster.ClientProvider
}

type Metadata struct {
	Labels      map[string]string
	Annotations map[string]string
}

func (v *DiskMetadataSetter) SetDiskMetadata(diskCID cpi.DiskCID, metadata Metadata) error {
	context, diskID := ParseDiskCID(diskCID)

	client, err := v.ClientProvider.New(context)
	if err != nil {
		return err
	}

	disk, err := client.PersistentVolumeClaims().Get("disk-" + diskID)
	if err != nil {
		return err
	}

	if disk.ObjectMeta.Labels == nil {
		disk.ObjectMeta.Labels = map[string]string{}
	}

	for k, v := range metadata.Labels {
		if k == "attached_at" {
			v = strings.Replace(v, ":", "_", -1)
		}

		k = "bosh.cloudfoundry.org/" + k
		errs := validation.IsQualifiedName(k)
		if len(errs) > 0 {
			return fmt.Errorf("Error setting disk metadata: label \"%s\": \"%s\": %s", k, v, strings.Join(errs, ": "))
		}

		errs = validation.IsValidLabelValue(v)
		if len(errs) > 0 {
			return fmt.Errorf("Error setting disk metadata: label \"%s\": \"%s\": %s", k, v, strings.Join(errs, ": "))
		}

		disk.ObjectMeta.Labels[k] = v
	}

	if disk.ObjectMeta.Annotations == nil {
		disk.ObjectMeta.Annotations = map[string]string{}
	}

	for k, v := range metadata.Annotations {
		errs := validation.IsQualifiedName(k)
		if len(errs) > 0 {
			return fmt.Errorf("Error setting disk metadata: annotation \"%s\": \"%s\": %s", k, v, strings.Join(errs, ": "))
		}

		disk.ObjectMeta.Annotations[k] = v
	}

	_, err = client.PersistentVolumeClaims().Update(disk)
	if err != nil {
		return err
	}

	return nil
}
