package fakes

import (
	"github.ibm.com/Bluemix/kubernetes-cpi/kubecluster"
	"k8s.io/client-go/kubernetes/fake"
	core "k8s.io/client-go/kubernetes/typed/core/v1"
	extensions "k8s.io/client-go/kubernetes/typed/extensions/v1beta1"
	"k8s.io/client-go/pkg/runtime"
	"k8s.io/client-go/testing"
)

//go:generate counterfeiter -o client_context.go --fake-name ClientContext . clientContext
type clientContext interface {
	Context() string
	Namespace() string
}

func NewClient(objects ...runtime.Object) *Client {
	return &Client{
		ClientContext: ClientContext{},
		Clientset:     *fake.NewSimpleClientset(objects...),
	}
}

var _ kubecluster.Client = NewClient()

// Client is a combination of a counterfeiter fake that exposes Namespace and Context
// and a Kubernetes fake.Clientset.
type Client struct {
	ClientContext
	fake.Clientset
}

func (c *Client) ConfigMaps() core.ConfigMapInterface {
	return c.Core().ConfigMaps(c.Namespace())
}

func (c *Client) Services() core.ServiceInterface {
	return c.Core().Services(c.Namespace())
}

func (c *Client) PersistentVolumeClaims() core.PersistentVolumeClaimInterface {
	return c.Core().PersistentVolumeClaims(c.Namespace())
}

func (c *Client) PersistentVolumes() core.PersistentVolumeInterface {
	return c.Core().PersistentVolumes()
}

func (c *Client) Pods() core.PodInterface {
	return c.Core().Pods(c.Namespace())
}

func (c *Client) Deployments() extensions.DeploymentInterface {
	return c.Extensions().Deployments(c.Namespace())
}

func (c *Client) IngressService() extensions.IngressInterface {
	return c.Extensions().Ingresses(c.Namespace())
}

func (c *Client) MatchingActions(verb, resource string) []testing.Action {
	result := []testing.Action{}
	for _, action := range c.Actions() {
		if action.Matches(verb, resource) {
			result = append(result, action)
		}
	}
	return result
}
