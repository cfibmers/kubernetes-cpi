package actions

import (
	"encoding/json"
	"strings"

	"github.ibm.com/Bluemix/kubernetes-cpi/cpi"
	"github.ibm.com/Bluemix/kubernetes-cpi/kubecluster"
	"k8s.io/client-go/pkg/api"
	"k8s.io/client-go/pkg/util/strategicpatch"
	"k8s.io/client-go/pkg/util/validation"

	bosherr "github.com/cloudfoundry/bosh-utils/errors"
)

type VMMetadataSetter struct {
	ClientProvider kubecluster.ClientProvider
}

func (v *VMMetadataSetter) SetVMMetadata(vmcid cpi.VMCID, metadata map[string]string) error {
	context, agentID := ParseVMCID(vmcid)

	client, err := v.ClientProvider.New(context)
	if err != nil {
		return bosherr.WrapError(err, "Creating a client")
	}

	pod, err := client.Pods().Get("agent-" + agentID)
	if err != nil {
		return bosherr.WrapError(err, "Getting pod")
	}

	old, err := json.Marshal(pod)
	if err != nil {
		return bosherr.WrapError(err, "Marshalling old pod")
	}

	for k, v := range metadata {
		k = "bosh.cloudfoundry.org/" + strings.ToLower(k)
		if len(validation.IsQualifiedName(k)) == 0 && len(validation.IsValidLabelValue(v)) == 0 {
			pod.ObjectMeta.Labels[k] = v
		}
	}

	new, err := json.Marshal(pod)
	if err != nil {
		return bosherr.WrapError(err, "Marshalling new pod")
	}

	patch, err := strategicpatch.CreateTwoWayMergePatch(old, new, pod)
	if err != nil {
		return bosherr.WrapError(err, "Creating TwoWayMergePatch")
	}

	_, err = client.Pods().Patch(pod.Name, api.StrategicMergePatchType, patch)
	if err != nil {
		return bosherr.WrapError(err, "Patching pod")
	}

	return nil
}
