package actions

import (
	"github.ibm.com/Bluemix/kubernetes-cpi/cpi"
	"github.ibm.com/Bluemix/kubernetes-cpi/kubecluster"
	"k8s.io/client-go/pkg/api/v1"
	"k8s.io/client-go/pkg/labels"

	bosherr "github.com/cloudfoundry/bosh-utils/errors"
)

type DiskFinder struct {
	ClientProvider kubecluster.ClientProvider
}

func (d *DiskFinder) HasDisk(diskCID cpi.DiskCID) (bool, error) {
	context, diskID := ParseDiskCID(diskCID)
	diskSelector, err := labels.Parse("bosh.cloudfoundry.org/disk-id=" + diskID)
	if err != nil {
		return false, bosherr.WrapError(err, "Parsing disk selector")
	}

	client, err := d.ClientProvider.New(context)
	if err != nil {
		return false, bosherr.WrapError(err, "Creating client")
	}

	listOptions := v1.ListOptions{LabelSelector: diskSelector.String()}
	pvcList, err := client.PersistentVolumeClaims().List(listOptions)
	if err != nil {
		return false, bosherr.WrapError(err, "Listing PVC options")
	}

	return len(pvcList.Items) > 0, nil
}
