package actions

import (
	"github.com/ScarletTanager/kubernetes-cpi/cpi"
	"github.com/ScarletTanager/kubernetes-cpi/kubecluster"
	"k8s.io/client-go/1.4/pkg/api"
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

	return client.PersistentVolumeClaims().Delete("disk-"+diskID, &api.DeleteOptions{GracePeriodSeconds: int64Ptr(0)})
}
