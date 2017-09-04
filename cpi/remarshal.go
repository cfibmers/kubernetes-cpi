package cpi

import (
	"encoding/json"
	bosherr "github.com/cloudfoundry/bosh-utils/errors"
)

func Remarshal(source interface{}, target interface{}) error {
	encoded, err := json.Marshal(source)
	if err != nil {
		return bosherr.WrapError(err, "Remarshalling source")
	}

	return json.Unmarshal(encoded, target)
}
