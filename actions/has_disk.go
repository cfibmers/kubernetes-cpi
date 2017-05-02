package actions

import (
	"github.com/ScarletTanager/kubernetes-cpi/cpi"
	"github.com/ScarletTanager/kubernetes-cpi/kubecluster"
	"k8s.io/client-go/pkg/api/v1"
	"k8s.io/client-go/pkg/labels"
)

type DiskFinder struct {
	ClientProvider kubecluster.ClientProvider
}

func (d *DiskFinder) HasDisk(diskCID cpi.DiskCID) (bool, error) {
	context, diskID := ParseDiskCID(diskCID)
	diskSelector, err := labels.Parse("bosh.cloudfoundry.org/disk-id=" + diskID)
	if err != nil {
		return false, err
	}

	client, err := d.ClientProvider.New(context)
	if err != nil {
		return false, err
	}

	listOptions := v1.ListOptions{LabelSelector: diskSelector.String()}
	pvcList, err := client.PersistentVolumeClaims().List(listOptions)
	if err != nil {
		return false, err
	}

	return len(pvcList.Items) > 0, nil
}
