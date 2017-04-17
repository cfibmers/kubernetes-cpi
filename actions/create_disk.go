package actions

import (
	"fmt"

	"github.com/ScarletTanager/kubernetes-cpi/cpi"
	"github.com/ScarletTanager/kubernetes-cpi/kubecluster"
	"k8s.io/client-go/1.4/pkg/api/resource"
	"k8s.io/client-go/1.4/pkg/api/v1"
)

type CreateDiskCloudProperties struct {
	Context string `json:"context"`
}

// DiskCreator simply creates a PersistentVolumeClaim. The attach process will
// turn the claim into a volume mounted into the pod.
type DiskCreator struct {
	ClientProvider    kubecluster.ClientProvider
	GUIDGeneratorFunc func() (string, error)
}

func (d *DiskCreator) CreateDisk(size uint, cloudProps CreateDiskCloudProperties, vmcid cpi.VMCID) (cpi.DiskCID, error) {
	diskID, err := d.GUIDGeneratorFunc()
	if err != nil {
		return "", err
	}

	volumeSize, err := resource.ParseQuantity(fmt.Sprintf("%dMi", size))
	if err != nil {
		return "", err
	}

	client, err := d.ClientProvider.New(cloudProps.Context)
	if err != nil {
		return "", err
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

	_, err = client.PersistentVolumeClaims().Create(&v1.PersistentVolumeClaim{
		ObjectMeta: v1.ObjectMeta{
			Name:      "disk-" + diskID,
			Namespace: client.Namespace(),
			Annotations: map[string]string{
				"volume.beta.kubernetes.io/storage-class": "ibmc-file-bronze",
			},
			Labels: map[string]string{
				"bosh.cloudfoundry.org/disk-id": diskID,
			},
		},
		Spec: v1.PersistentVolumeClaimSpec{
			// VolumeName:  volumeName,
			AccessModes: []v1.PersistentVolumeAccessMode{v1.ReadWriteOnce},
			Resources: v1.ResourceRequirements{
				Requests: v1.ResourceList{
					v1.ResourceStorage: volumeSize,
				},
			},
		},
	})

	if err != nil {
		return "", err
	}

	return NewDiskCID(client.Context(), diskID), nil
}
