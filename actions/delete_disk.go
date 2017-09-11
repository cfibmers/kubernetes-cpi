package actions

import (
	"github.ibm.com/Bluemix/kubernetes-cpi/cpi"
	"github.ibm.com/Bluemix/kubernetes-cpi/kubecluster"
	"k8s.io/client-go/pkg/api/v1"

	bosherr "github.com/cloudfoundry/bosh-utils/errors"
)

type DiskDeleter struct {
	ClientProvider kubecluster.ClientProvider
}

func (d *DiskDeleter) DeleteDisk(diskCID cpi.DiskCID) error {
	context, diskID := ParseDiskCID(diskCID)
	client, err := d.ClientProvider.New(context)
	if err != nil {
		return bosherr.WrapError(err, "Creating client")
	}

	return client.PersistentVolumeClaims().Delete("disk-"+diskID, &v1.DeleteOptions{GracePeriodSeconds: int64Ptr(0)})
}
