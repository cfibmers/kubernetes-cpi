package actions

import (
	"github.ibm.com/Bluemix/kubernetes-cpi/cpi"
	"github.ibm.com/Bluemix/kubernetes-cpi/kubecluster"
	"k8s.io/client-go/pkg/api/v1"
)

type DiskDeleter struct {
	ClientProvider kubecluster.ClientProvider
}

func (d *DiskDeleter) DeleteDisk(diskCID cpi.DiskCID) error {
	context, diskID := ParseDiskCID(diskCID)
	client, err := d.ClientProvider.New(context)
	if err != nil {
		return err
	}

	return client.PersistentVolumeClaims().Delete("disk-"+diskID, &v1.DeleteOptions{GracePeriodSeconds: int64Ptr(0)})
}
