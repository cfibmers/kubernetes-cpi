package actions

import "github.ibm.com/Bluemix/kubernetes-cpi/cpi"

type StemcellCloudProperties struct {
	Image string `json:"image"`
}

func CreateStemcell(image string, cloudProps StemcellCloudProperties) (cpi.StemcellCID, error) {
	return cpi.StemcellCID(cloudProps.Image), nil
}

func DeleteStemcell(stemcellCID cpi.StemcellCID) error {
	return nil
}
