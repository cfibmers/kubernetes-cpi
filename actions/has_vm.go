package actions

import (
	"github.ibm.com/Bluemix/kubernetes-cpi/cpi"
	"github.ibm.com/Bluemix/kubernetes-cpi/kubecluster"
	"k8s.io/client-go/pkg/api/v1"
	"k8s.io/client-go/pkg/labels"
)

type VMFinder struct {
	ClientProvider kubecluster.ClientProvider
}

func (f *VMFinder) HasVM(vmcid cpi.VMCID) (bool, error) {
	_, pod, err := f.FindVM(vmcid)
	return pod != nil, err
}

func (f *VMFinder) FindVM(vmcid cpi.VMCID) (string, *v1.Pod, error) {
	context, agentID := ParseVMCID(vmcid)
	agentSelector, err := labels.Parse("bosh.cloudfoundry.org/agent-id=" + agentID)
	if err != nil {
		return "", nil, err
	}

	client, err := f.ClientProvider.New(context)
	if err != nil {
		return "", nil, err
	}

	listOptions := v1.ListOptions{LabelSelector: agentSelector.String()}
	podList, err := client.Pods().List(listOptions)
	if err != nil {
		return "", nil, err
	}

	if len(podList.Items) > 0 {
		return context, &podList.Items[0], nil
	}

	return "", nil, nil
}
