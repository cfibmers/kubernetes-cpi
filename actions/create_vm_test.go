package actions_test

import (
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"os"

	kubeerrors "k8s.io/client-go/pkg/api/errors"
	"k8s.io/client-go/pkg/api/resource"
	"k8s.io/client-go/pkg/api/unversioned"
	"k8s.io/client-go/pkg/api/v1"
	"k8s.io/client-go/pkg/apis/extensions/v1beta1"
	"k8s.io/client-go/pkg/runtime"
	"k8s.io/client-go/pkg/util/intstr"
	"k8s.io/client-go/testing"

	"github.ibm.com/Bluemix/kubernetes-cpi/actions"
	"github.ibm.com/Bluemix/kubernetes-cpi/agent"
	"github.ibm.com/Bluemix/kubernetes-cpi/config"
	"github.ibm.com/Bluemix/kubernetes-cpi/cpi"
	"github.ibm.com/Bluemix/kubernetes-cpi/kubecluster/fakes"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("CreateVM", func() {
	var (
		fakeClient   *fakes.Client
		fakeProvider *fakes.ClientProvider
		agentConf    *config.Agent

		agentID  string
		env      cpi.Environment
		networks cpi.Networks

		vmCreator *actions.VMCreator
	)

	BeforeEach(func() {
		fakeClient = fakes.NewClient()
		fakeClient.ContextReturns("bosh")
		fakeClient.NamespaceReturns("bosh-namespace")

		fakeProvider = &fakes.ClientProvider{}
		fakeProvider.NewReturns(fakeClient, nil)

		agentConf = &config.Agent{
			Blobstore:  "some-blobstore-config",
			MessageBus: "message-bus-url",
			NTPServers: []string{"1.example.org", "2.example.org"},
		}

		vmCreator = &actions.VMCreator{
			ClientProvider: fakeProvider,
			AgentConfig:    agentConf,
		}

		agentID = "agent-id"
		env = cpi.Environment{"passed": "along"}
		networks = cpi.Networks{
			"dynamic-network": cpi.Network{
				Type: "dynamic",
				DNS:  []string{"8.8.8.8", "8.8.4.4"},
				CloudProperties: map[string]interface{}{
					"dynamic-key": "dynamic-value",
				},
			},
		}
	})

	Describe("Create", func() {
		var (
			stemcellCID    cpi.StemcellCID
			cloudProps     actions.VMCloudProperties
			diskCIDs       []cpi.DiskCID
			secretTypeMap map[v1.SecretType]string
			tmpFile        string
		)

		BeforeEach(func() {
			stemcellCID = cpi.StemcellCID("ScarletTanager/kubernetes-stemcell:999")
			cloudProps = actions.VMCloudProperties{Context: "bosh"}
			diskCIDs = []cpi.DiskCID{}
		})

		It("returns a VM Cloud ID", func() {
			vmcid, err := vmCreator.Create(agentID, stemcellCID, cloudProps, networks, diskCIDs, env)
			Expect(err).NotTo(HaveOccurred())
			Expect(vmcid).To(Equal(actions.NewVMCID("bosh", agentID)))
		})

		It("gets a client with the context from the cloud properties", func() {
			_, err := vmCreator.Create(agentID, stemcellCID, cloudProps, networks, diskCIDs, env)
			Expect(err).NotTo(HaveOccurred())

			Expect(fakeProvider.NewCallCount()).To(Equal(1))
			Expect(fakeProvider.NewArgsForCall(0)).To(Equal("bosh"))
		})

		Context("when replicas property is present in cloud properties", func() {

			BeforeEach(func() {
				stemcellCID = cpi.StemcellCID("ScarletTanager/kubernetes-stemcell:999")
				cloudProps = actions.VMCloudProperties{Context: "bosh"}
				diskCIDs = []cpi.DiskCID{}
			})

			testReplicaCount := func(val int32, shouldError bool) {
				cloudProps.Replicas = &val
				_, err := vmCreator.Create(agentID, stemcellCID, cloudProps, networks, diskCIDs, env)
				if shouldError {
					Expect(err).To(HaveOccurred())
				} else {

					matches := fakeClient.MatchingActions("create", "deployments")
					Expect(matches).To(HaveLen(1))

					deployment := matches[0].(testing.CreateAction).GetObject().(*v1beta1.Deployment)
					Expect(deployment.Name).To(Equal("agent-" + agentID))
					Expect(deployment.Annotations).To(BeEmpty())
					Expect(deployment.Spec.Replicas).To(Equal(cloudProps.Replicas))
					Expect(deployment.Spec.Template.Spec.Hostname).To(Equal(agentID))
					Expect(*deployment.Spec.ProgressDeadlineSeconds).To(Equal(actions.ProgressDeadlineSeconds))

					Expect(err).ToNot(HaveOccurred())
				}
			}

			It("evaluates the replicas property on failure", func() {
				testReplicaCount(int32(-2), true)
				testReplicaCount(int32(0), true)
			})

			It("evaluates the replicas property for one replica", func() {
				testReplicaCount(int32(1), false)
			})

			It("evaluates the replicas property for more than one replica", func() {
				testReplicaCount(int32(2), false)
			})
		})

		Context("when getting the client fails", func() {
			BeforeEach(func() {
				fakeProvider.NewReturns(nil, errors.New("boom"))
			})

			It("gets a client for the appropriate context", func() {
				_, err := vmCreator.Create(agentID, stemcellCID, cloudProps, networks, diskCIDs, env)
				Expect(err).To(MatchError("boom"))
			})
		})

		It("creates the target namespace", func() {
			_, err := vmCreator.Create(agentID, stemcellCID, cloudProps, networks, diskCIDs, env)
			Expect(err).NotTo(HaveOccurred())

			matches := fakeClient.MatchingActions("create", "namespaces")
			Expect(matches).To(HaveLen(1))

			namespace := matches[0].(testing.CreateAction).GetObject().(*v1.Namespace)
			Expect(namespace.Name).To(Equal("bosh-namespace"))
		})

		Context("when the namespace already exists", func() {
			BeforeEach(func() {
				fakeClient = fakes.NewClient(
					&v1.Namespace{ObjectMeta: v1.ObjectMeta{Name: "bosh-namespace"}},
				)
				fakeClient.ContextReturns("bosh")
				fakeClient.NamespaceReturns("bosh-namespace")
				fakeProvider.NewReturns(fakeClient, nil)
			})

			It("skips namespace creation", func() {
				_, err := vmCreator.Create(agentID, stemcellCID, cloudProps, networks, diskCIDs, env)
				Expect(err).NotTo(HaveOccurred())

				Expect(fakeClient.MatchingActions("get", "namespaces")).To(HaveLen(1))
				Expect(fakeClient.MatchingActions("create", "namespaces")).To(HaveLen(0))
			})
		})

		Context("when the namespace create fails with StatusReasonAlreadyExists", func() {
			BeforeEach(func() {
				fakeClient.PrependReactor("create", "namespaces", func(action testing.Action) (bool, runtime.Object, error) {
					gr := unversioned.GroupResource{Group: "", Resource: "namespaces"}
					return true, nil, kubeerrors.NewAlreadyExists(gr, "bosh-namespace")
				})
			})

			It("keeps calm and carries on", func() {
				_, err := vmCreator.Create(agentID, stemcellCID, cloudProps, networks, diskCIDs, env)
				Expect(err).NotTo(HaveOccurred())

				Expect(fakeClient.MatchingActions("get", "namespaces")).To(HaveLen(1))
				Expect(fakeClient.MatchingActions("create", "namespaces")).To(HaveLen(1))
			})
		})

		Context("when the namespace create fails", func() {
			BeforeEach(func() {
				fakeClient.PrependReactor("create", "namespaces", func(action testing.Action) (bool, runtime.Object, error) {
					return true, nil, errors.New("namespace-welp")
				})
			})

			It("returns an error", func() {
				_, err := vmCreator.Create(agentID, stemcellCID, cloudProps, networks, diskCIDs, env)
				Expect(err).To(MatchError("namespace-welp"))
				Expect(fakeClient.MatchingActions("create", "namespaces")).To(HaveLen(1))
			})
		})

		Context("when no networks are defined", func() {
			BeforeEach(func() {
				networks = cpi.Networks{}
			})

			It("returns an error", func() {
				_, err := vmCreator.Create(agentID, stemcellCID, cloudProps, networks, diskCIDs, env)
				Expect(err).To(MatchError("a network is required"))
			})
		})

		Context("when multiple networks are defined", func() {
			BeforeEach(func() {
				networks = cpi.Networks{
					"manual-network": cpi.Network{
						Type:    "manual",
						IP:      "1.2.3.4",
						Netmask: "255.255.0.0",
						Gateway: "1.2.0.1",
						DNS:     []string{"8.8.8.8", "8.8.4.4"},
						Default: []string{"dns", "gateway"},
						CloudProperties: map[string]interface{}{
							"key": "value",
						},
					},
					"dynamic-network": cpi.Network{
						Type: "dynamic",
						DNS:  []string{"8.8.8.8", "8.8.4.4"},
						CloudProperties: map[string]interface{}{
							"dynamic-key": "dynamic-value",
						},
					},
				}
			})

			It("returns an error", func() {
				_, err := vmCreator.Create(agentID, stemcellCID, cloudProps, networks, diskCIDs, env)
				Expect(err).To(MatchError("multiple networks not supported"))
			})
		})

		It("creates the config map for agent settings", func() {
			_, err := vmCreator.Create(agentID, stemcellCID, cloudProps, networks, diskCIDs, env)
			Expect(err).NotTo(HaveOccurred())

			matches := fakeClient.MatchingActions("create", "configmaps")
			Expect(matches).To(HaveLen(1))

			instanceSettings, err := vmCreator.InstanceSettings(agentID, networks, env)
			Expect(err).NotTo(HaveOccurred())
			instanceJSON, err := json.Marshal(instanceSettings)
			Expect(err).NotTo(HaveOccurred())

			configMap := matches[0].(testing.CreateAction).GetObject().(*v1.ConfigMap)
			Expect(configMap.Name).To(Equal("agent-" + agentID))
			Expect(configMap.Labels["bosh.cloudfoundry.org/agent-id"]).To(Equal(agentID))
			Expect(configMap.Data["instance_settings"]).To(MatchJSON(instanceJSON))
		})

		Context("when the config map create fails", func() {
			BeforeEach(func() {
				fakeClient.PrependReactor("create", "configmaps", func(action testing.Action) (bool, runtime.Object, error) {
					return true, nil, errors.New("configmap-welp")
				})
			})

			It("returns an error", func() {
				_, err := vmCreator.Create(agentID, stemcellCID, cloudProps, networks, diskCIDs, env)
				Expect(err).To(MatchError("configmap-welp"))
				Expect(fakeClient.MatchingActions("create", "configmaps")).To(HaveLen(1))
			})
		})

		Context("when service definitions are present in the cloud properties", func() {
			BeforeEach(func() {
				cloudProps.Services = []actions.Service{
					{
						Name: "director",
						Type: "NodePort",
						Ports: []actions.Port{
							{Name: "agent", Protocol: "TCP", Port: 6868, NodePort: 32068},
							{Name: "director", Protocol: "TCP", Port: 25555, NodePort: 32067},
						},
					},
					{
						Name:      "blobstore",
						ClusterIP: "10.0.0.1",
						Ports: []actions.Port{
							{Port: 25250, Protocol: "TCP"},
						},
					},
					{
						Name:      "bosh-dns",
						Type:      "LoadBalancer",
						ClusterIP: "10.0.0.2",
						Ports: []actions.Port{
							{Name: "bosh-dns", Protocol: "TCP", Port: 53, NodePort: 32069},
						},
					},
					{
						Name: "bosh-dns-1",
						Type: "LoadBalancer",
						Ports: []actions.Port{
							{Name: "bosh-dns-1", Protocol: "TCP", Port: 53, NodePort: 32070},
						},
					},
					{
						Name: "ha-proxy-80",
						Type: "LoadBalancer",
						Ports: []actions.Port{
							{Name: "ha-proxy-80", Protocol: "TCP", Port: 80, NodePort: 30080, TargetPort: 80},
						},
						Selector:       map[string]string{"bosh.cloudfoundry.org/job": "ha_proxy_z1"},
						LoadBalancerIP: "169.10.10.10",
					},
					{
						Name: "ha-proxy-443",
						Type: "NodePort",
						Ports: []actions.Port{
							{Name: "ha-proxy-443", Protocol: "TCP", Port: 443, NodePort: 30443, TargetPort: 443},
						},
						Selector:    map[string]string{"bosh.cloudfoundry.org/job": "ha_proxy_z1"},
						ExternalIPs: []string{"158.10.10.10", "158.10.10.11"},
					},
					{
						Name: "nginx",
						Selector: map[string]string{
							"app": "nginx",
						},
						Ports: []actions.Port{
							{Port: 80},
						},
					},
					{
						Name: "ingress1",
						Type: "Ingress",
						Backend: &v1beta1.IngressBackend{
							ServiceName: "nginx",
							ServicePort: intstr.FromInt(80),
						},
					},
					{
						Name: "ingress2",
						Type: "Ingress",
						TLS: []v1beta1.IngressTLS{
							{
								Hosts:      []string{"apoorv-dev3.eu-central.containers.mybluemix.net"},
								SecretName: "apoorv-dev3",
							},
						},
						Rules: []v1beta1.IngressRule{
							{
								Host: "apoorv-dev3.eu-central.containers.mybluemix.net",
								IngressRuleValue: v1beta1.IngressRuleValue{
									HTTP: &v1beta1.HTTPIngressRuleValue{
										Paths: []v1beta1.HTTPIngressPath{
											{
												Path: "/",
												Backend: v1beta1.IngressBackend{
													ServiceName: "nginx",
													ServicePort: intstr.FromInt(80),
												},
											},
										},
									},
								},
							},
						},
					},
				}
			})

			It("creates the services", func() {
				_, err := vmCreator.Create(agentID, stemcellCID, cloudProps, networks, diskCIDs, env)
				Expect(err).NotTo(HaveOccurred())

				matches := fakeClient.MatchingActions("create", "services")
				Expect(matches).To(HaveLen(7))

				for i, s := range matches {
					service := s.(testing.CreateAction).GetObject().(*v1.Service)
					expected := cloudProps.Services[i]
					Expect(service.Name).To(Equal(expected.Name))
					Expect(service.Labels["bosh.cloudfoundry.org/agent-id"]).To(Equal(agentID))
					if expected.Type != "" {
						Expect(service.Spec.Type).To(Equal(v1.ServiceType(expected.Type)))
					} else {
						Expect(service.Spec.Type).To(Equal(v1.ServiceTypeClusterIP))
					}

					if expected.Selector != nil {
						Expect(service.Spec.Selector).To(Equal(expected.Selector))
					}

					if len(expected.Ports) > 0 {
						var ports []v1.ServicePort

						for _, p := range expected.Ports {
							ports = append(ports, v1.ServicePort{Name: p.Name, Protocol: v1.Protocol(p.Protocol), Port: p.Port, NodePort: p.NodePort, TargetPort: intstr.FromInt(p.TargetPort)})
						}

						Expect(service.Spec.Ports).To(Equal(ports))
					}

					Expect(service.Spec.LoadBalancerIP).To(Equal(expected.LoadBalancerIP))
					Expect(service.Spec.ExternalIPs).To(Equal(expected.ExternalIPs))
				}

				omatches := fakeClient.MatchingActions("create", "ingresses")
				fmt.Println(len(omatches))
				Expect(omatches).To(HaveLen(2))

				fmt.Println(omatches[0].GetResource())
				iService := omatches[0].(testing.CreateAction).GetObject().(*v1beta1.Ingress)

				Expect(iService.Name).To(Equal("ingress1"))
				Expect(*iService.Spec.Backend).To(Equal(
					v1beta1.IngressBackend{ServiceName: "nginx", ServicePort: intstr.FromInt(80)},
				))

				iService = omatches[1].(testing.CreateAction).GetObject().(*v1beta1.Ingress)
				Expect(iService.Name).To(Equal("ingress2"))
				Expect(iService.Spec.TLS).To(ConsistOf(
					v1beta1.IngressTLS{
						Hosts:      []string{"apoorv-dev3.eu-central.containers.mybluemix.net"},
						SecretName: "apoorv-dev3",
					},
				))
				Expect(iService.Spec.Rules).To(ConsistOf(
					v1beta1.IngressRule{
						Host: "apoorv-dev3.eu-central.containers.mybluemix.net",
						IngressRuleValue: v1beta1.IngressRuleValue{
							HTTP: &v1beta1.HTTPIngressRuleValue{
								Paths: []v1beta1.HTTPIngressPath{
									{
										Path: "/",
										Backend: v1beta1.IngressBackend{
											ServiceName: "nginx",
											ServicePort: intstr.FromInt(80),
										},
									},
								},
							},
						},
					},
				))
			})

			Context("when the service create fails", func() {
				BeforeEach(func() {
					fakeClient.PrependReactor("create", "services", func(action testing.Action) (bool, runtime.Object, error) {
						return true, nil, errors.New("service-welp")
					})
				})

				It("returns an error", func() {
					_, err := vmCreator.Create(agentID, stemcellCID, cloudProps, networks, diskCIDs, env)
					Expect(err).To(MatchError("service-welp"))
					Expect(fakeClient.MatchingActions("create", "services")).To(HaveLen(1))
				})
			})
		})

		Context("when secret definitions are present in the cloud properties", func() {
			BeforeEach(func() {
				file, err := ioutil.TempFile(os.TempDir(), ".dockercfg")
				tmpFile = file.Name()
				Expect(err).ToNot(HaveOccurred())

				cloudProps.Secrets = []actions.Secret{
					{
						Name: "secret-defaultType",
						Data: map[string]string{
							"username": "admin",
							"password": "admin",
						},
						StringData: map[string]string{
							"foo": "bar",
						},
					},
					{
						Name: "secret-TLS",
						Type: "TLS",
						Data: map[string]string{
							"tls.key":  "fake-key",
							"tls.cert": "fake-cert",
						},
					},
					{
						Name: "secret-ServiceAccountToken",
						Type: "ServiceAccountToken",
						Data: map[string]string{
							"token": "fake-token",
						},
						Annotations: map[string]string{
							"kubernetes.io/service-account.name": "fake-account-name",
							"kubernetes.io/service-account.uid":  "fake-account-uid",
						},
					},
					{
						Name: "secret-DockerCfg",
						Type: "DockerCfg",
						Data: map[string]string{
							".dockercfg": tmpFile,
						},
					},
				}
				secretTypeMap = map[v1.SecretType]string{
					v1.SecretTypeOpaque:              "Opaque",
					v1.SecretTypeTLS:                 "TLS",
					v1.SecretTypeServiceAccountToken: "ServiceAccountToken",
					v1.SecretTypeDockercfg:           "DockerCfg",
				}
			})

			AfterEach(func() {
				os.Remove(tmpFile)
			})

			It("creates the secret", func() {
				_, err := vmCreator.Create(agentID, stemcellCID, cloudProps, networks, diskCIDs, env)
				Expect(err).NotTo(HaveOccurred())

				matches := fakeClient.MatchingActions("create", "secrets")
				Expect(matches).To(HaveLen(4))

				for i, s := range matches {
					secret := s.(testing.CreateAction).GetObject().(*v1.Secret)
					expected := cloudProps.Secrets[i]
					Expect(secret.Name).To(Equal(expected.Name))
					Expect(secret.Labels["bosh.cloudfoundry.org/agent-id"]).To(Equal(agentID))
					if expected.Type != "" {
						Expect(secretTypeMap[secret.Type]).To(Equal(expected.Type))
					} else {
						Expect(secretTypeMap[secret.Type]).To(Equal("Opaque"))
					}
				}
			})

			Context("when the secret create fails", func() {
				BeforeEach(func() {
					fakeClient.PrependReactor("create", "secrets", func(action testing.Action) (bool, runtime.Object, error) {
						return true, nil, errors.New("secret-welp")
					})
				})

				It("returns an error", func() {
					_, err := vmCreator.Create(agentID, stemcellCID, cloudProps, networks, diskCIDs, env)
					Expect(err).To(MatchError("secret-welp"))
					Expect(fakeClient.MatchingActions("create", "secrets")).To(HaveLen(1))
				})
			})
		})

		It("creates a pod", func() {
			_, err := vmCreator.Create(agentID, stemcellCID, cloudProps, networks, diskCIDs, env)
			Expect(err).NotTo(HaveOccurred())

			matches := fakeClient.MatchingActions("create", "pods")
			Expect(matches).To(HaveLen(1))

			trueValue := true
			rootUID := int64(0)

			pod := matches[0].(testing.CreateAction).GetObject().(*v1.Pod)
			Expect(pod.Name).To(Equal("agent-" + agentID))
			Expect(pod.Annotations).To(BeEmpty())
			Expect(pod.Labels["bosh.cloudfoundry.org/agent-id"]).To(Equal(agentID))
			Expect(pod.Spec.Hostname).To(Equal(agentID))
			Expect(pod.Spec.Containers).To(ConsistOf(
				v1.Container{
					Name:            "bosh-job",
					Image:           "ScarletTanager/kubernetes-stemcell:999",
					ImagePullPolicy: v1.PullAlways,
					Command:         []string{"/usr/sbin/runsvdir-start"},
					Args:            []string{},
					SecurityContext: &v1.SecurityContext{
						Privileged: &trueValue,
						RunAsUser:  &rootUID,
					},
					VolumeMounts: []v1.VolumeMount{{
						Name:      "bosh-config",
						MountPath: "/var/vcap/bosh/instance_settings.json",
						SubPath:   "instance_settings.json",
					}, {
						Name:      "bosh-ephemeral",
						MountPath: "/var/vcap/data",
					}},
				}))

			Expect(pod.Spec.Volumes).To(ConsistOf(
				v1.Volume{
					Name: "bosh-config",
					VolumeSource: v1.VolumeSource{
						ConfigMap: &v1.ConfigMapVolumeSource{
							LocalObjectReference: v1.LocalObjectReference{
								Name: "agent-" + agentID,
							},
							Items: []v1.KeyToPath{{
								Key:  "instance_settings",
								Path: "instance_settings.json",
							}},
						},
					},
				},
				v1.Volume{
					Name: "bosh-ephemeral",
					VolumeSource: v1.VolumeSource{
						EmptyDir: &v1.EmptyDirVolumeSource{},
					},
				}))
		})

		Context("when the network contains an IP", func() {
			BeforeEach(func() {
				networks = cpi.Networks{
					"manual-network": cpi.Network{
						Type:    "manual",
						IP:      "1.2.3.4",
						Netmask: "255.255.0.0",
						Gateway: "1.2.0.1",
						DNS:     []string{"8.8.8.8", "8.8.4.4"},
						Default: []string{"dns", "gateway"},
						CloudProperties: map[string]interface{}{
							"key": "value",
						},
					},
				}
			})

			It("annotates the pod with the IP address information", func() {
				_, err := vmCreator.Create(agentID, stemcellCID, cloudProps, networks, diskCIDs, env)
				Expect(err).NotTo(HaveOccurred())

				matches := fakeClient.MatchingActions("create", "pods")
				Expect(matches).To(HaveLen(1))

				pod := matches[0].(testing.CreateAction).GetObject().(*v1.Pod)
				Expect(pod.Annotations["bosh.cloudfoundry.org/ip-address"]).To(Equal("1.2.3.4"))
			})
		})

		Context("when resource definitions are present in the cloud properties", func() {
			BeforeEach(func() {
				cloudProps.Resources = actions.Resources{
					Limits: actions.ResourceList{
						actions.ResourceMemory: "1Gi",
						actions.ResourceCPU:    "500m",
					},
					Requests: actions.ResourceList{
						actions.ResourceMemory: "64Mi",
						actions.ResourceCPU:    "100m",
					},
				}
			})

			It("sets resource limits and requests on the Pod", func() {
				_, err := vmCreator.Create(agentID, stemcellCID, cloudProps, networks, diskCIDs, env)
				Expect(err).NotTo(HaveOccurred())

				matches := fakeClient.MatchingActions("create", "pods")
				Expect(matches).To(HaveLen(1))

				pod := matches[0].(testing.CreateAction).GetObject().(*v1.Pod)
				Expect(pod.Spec.Containers[0].Resources).To(Equal(v1.ResourceRequirements{
					Requests: v1.ResourceList{
						v1.ResourceMemory: resource.MustParse("64Mi"),
						v1.ResourceCPU:    resource.MustParse("100m"),
					},
					Limits: v1.ResourceList{
						v1.ResourceMemory: resource.MustParse("1Gi"),
						v1.ResourceCPU:    resource.MustParse("500m"),
					},
				}))
			})

			Context("when a resource request quantity cannot be parsed", func() {
				BeforeEach(func() {
					cloudProps.Resources = actions.Resources{
						Requests: actions.ResourceList{actions.ResourceMemory: "12nuts"},
						Limits:   actions.ResourceList{actions.ResourceMemory: "1Gi"},
					}
				})

				It("returns an error", func() {
					_, err := vmCreator.Create(agentID, stemcellCID, cloudProps, networks, diskCIDs, env)
					Expect(err).To(MatchError(ContainSubstring("quantities must match the regular expression")))
				})
			})

			Context("when resource limit quantity cannot be parsed", func() {
				BeforeEach(func() {
					cloudProps.Resources = actions.Resources{
						Requests: actions.ResourceList{actions.ResourceMemory: "1Gi"},
						Limits:   actions.ResourceList{actions.ResourceMemory: "12nuts"},
					}
				})

				It("returns an error", func() {
					_, err := vmCreator.Create(agentID, stemcellCID, cloudProps, networks, diskCIDs, env)
					Expect(err).To(MatchError(ContainSubstring("quantities must match the regular expression")))
				})
			})

			Context("when an unsupported resource type is specified", func() {
				BeforeEach(func() {
					cloudProps.Resources = actions.Resources{
						Requests: actions.ResourceList{"goo": "1Gi"},
					}
				})

				It("returns an error", func() {
					_, err := vmCreator.Create(agentID, stemcellCID, cloudProps, networks, diskCIDs, env)
					Expect(err).To(MatchError("goo is not a supported resource type"))
				})
			})
		})

		Context("when creating the pod fails", func() {
			BeforeEach(func() {
				fakeClient.PrependReactor("create", "pods", func(action testing.Action) (bool, runtime.Object, error) {
					return true, nil, errors.New("pods-welp")
				})
			})

			It("returns an error", func() {
				_, err := vmCreator.Create(agentID, stemcellCID, cloudProps, networks, diskCIDs, env)
				Expect(err).To(MatchError("pods-welp"))
				Expect(fakeClient.MatchingActions("create", "pods")).To(HaveLen(1))
			})
		})
	})

	Describe("InstanceSettings", func() {
		It("copies the blobstore from the agent config", func() {
			agentSettings, err := vmCreator.InstanceSettings(agentID, networks, env)
			Expect(err).NotTo(HaveOccurred())
			Expect(agentSettings.Blobstore).To(Equal(agentConf.Blobstore))
		})

		It("copies the message bus from the agent config", func() {
			agentSettings, err := vmCreator.InstanceSettings(agentID, networks, env)
			Expect(err).NotTo(HaveOccurred())
			Expect(agentSettings.MessageBus).To(Equal(agentConf.MessageBus))
		})

		It("copies the ntp servers from the agent config", func() {
			agentSettings, err := vmCreator.InstanceSettings(agentID, networks, env)
			Expect(err).NotTo(HaveOccurred())
			Expect(agentSettings.NTPServers).To(Equal(agentConf.NTPServers))
		})

		It("sets the agent ID", func() {
			agentSettings, err := vmCreator.InstanceSettings(agentID, networks, env)
			Expect(err).NotTo(HaveOccurred())
			Expect(agentSettings.AgentID).To(Equal(agentID))
		})

		It("sets the VM name", func() {
			agentSettings, err := vmCreator.InstanceSettings(agentID, networks, env)
			Expect(err).NotTo(HaveOccurred())
			Expect(agentSettings.VM).To(Equal(agent.VM{Name: agentID}))
		})

		It("propagates the bosh environment", func() {
			agentSettings, err := vmCreator.InstanceSettings(agentID, networks, env)
			Expect(err).NotTo(HaveOccurred())
			Expect(agentSettings.Env).To(Equal(env))
		})

		It("generates an empty persistent disk map by default", func() {
			agentSettings, err := vmCreator.InstanceSettings(agentID, networks, env)
			Expect(err).NotTo(HaveOccurred())
			Expect(agentSettings.Disks).To(Equal(agent.Disks{}))
		})

		It("sets the network configuration", func() {
			agentSettings, err := vmCreator.InstanceSettings(agentID, networks, env)
			Expect(err).NotTo(HaveOccurred())
			Expect(agentSettings.Networks).To(Equal(agent.Networks{
				"dynamic-network": agent.Network{
					Type:          "dynamic",
					Preconfigured: true,
					DNS: []string{
						"8.8.8.8",
						"8.8.4.4",
					},
				},
			}))
		})

		Context("when the networks fails to remarshal", func() {
			BeforeEach(func() {
				networks["dynamic-network"].CloudProperties["channel"] = make(chan struct{})
			})

			It("returns an error", func() {
				_, err := vmCreator.InstanceSettings(agentID, networks, env)
				Expect(err).To(HaveOccurred())
				Expect(err).To(BeAssignableToTypeOf(&json.UnsupportedTypeError{}))
			})
		})
	})
})
