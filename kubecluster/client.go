package kubecluster

import (
	"k8s.io/client-go/kubernetes"
	core "k8s.io/client-go/kubernetes/typed/core/v1"
	v1beta1 "k8s.io/client-go/kubernetes/typed/extensions/v1beta1"
)

type Client interface {
	Context() string
	Namespace() string

	Core() core.CoreV1Interface

	ConfigMaps() core.ConfigMapInterface
	PersistentVolumes() core.PersistentVolumeInterface
	PersistentVolumeClaims() core.PersistentVolumeClaimInterface
	Pods() core.PodInterface
	Deployments() v1beta1.DeploymentInterface
	Services() core.ServiceInterface
}

type client struct {
	context   string
	namespace string

	*kubernetes.Clientset
}

var _ Client = &client{}

func (c *client) Context() string {
	return c.context
}

func (c *client) Namespace() string {
	return c.namespace
}

func (c *client) ConfigMaps() core.ConfigMapInterface {
	return c.Core().ConfigMaps(c.namespace)
}

func (c *client) PersistentVolumeClaims() core.PersistentVolumeClaimInterface {
	return c.Core().PersistentVolumeClaims(c.namespace)
}

func (c *client) PersistentVolumes() core.PersistentVolumeInterface {
	return c.Core().PersistentVolumes()
}

func (c *client) Pods() core.PodInterface {
	return c.Core().Pods(c.namespace)
}

func (c *client) Deployments() v1beta1.DeploymentInterface {
	return c.Extensions().Deployments(c.namespace)
}

func (c *client) Services() core.ServiceInterface {
	return c.Core().Services(c.namespace)
}
