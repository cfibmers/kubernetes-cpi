package vm_test

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	testHelper "github.ibm.com/Bluemix/kubernetes-cpi/integration/test_assets"
	"k8s.io/client-go/pkg/apis/extensions/v1beta1"
	"k8s.io/client-go/pkg/util/intstr"
)

const configPath = "integration/test_assets/cpi_methods/config.json"
const agentPath = "integration/test_assets/cpi_methods/agent.json"

var _ = Describe("Creating a VM", func() {
	var (
		err                             error
		jsonPayload                     string
		clusterName                     string
		kubeConfig                      string
		rootTemplatePath, tmpConfigPath string
		replacementMap                  map[string]string
		errorOutput                     map[string]interface{}
		resultOutput                    map[string]interface{}

		agentId string
	)

	BeforeEach(func() {
		clusterName = os.Getenv("CLUSTER_NAME")
		Expect(err).ToNot(HaveOccurred())

		kubeConfig = os.Getenv("KUBECONFIG")
		Expect(err).ToNot(HaveOccurred())

		// This assumes you are in a certain directory - change?
		pwd, _ := os.Getwd()
		rootTemplatePath = filepath.Join(pwd, "..", "..")

		agentId = "490c18a5-3bb4-4b92-8550-ee4a1e955624"
		replacementMap = map[string]string{
			"agentID": agentId,
			"context": clusterName,
		}

		tmpConfigPath, err = testHelper.CreateTmpConfigFile(rootTemplatePath, configPath, kubeConfig)
		Expect(err).ToNot(HaveOccurred())

		numberOfPods, err := testHelper.PodCount("integration")
		Expect(err).NotTo(HaveOccurred())
		Expect(numberOfPods).To(Equal(0))

	})

	AfterEach(func() {
		err = os.Remove(tmpConfigPath)
		Expect(err).ToNot(HaveOccurred())
	})

	Context("Creating a VM", func() {
		var numberOfPods int

		BeforeEach(func() {
			jsonPayload, err = testHelper.GenerateCpiJsonPayload("create_vm", rootTemplatePath, replacementMap)
			Expect(err).ToNot(HaveOccurred())

			numberOfPods, err = testHelper.PodCount("integration")
			Expect(err).NotTo(HaveOccurred())
			Expect(numberOfPods).To(Equal(0))
		})

		AfterEach(func() {
			deleteAll := exec.Command("kubectl", "-n", "integration", "delete", "po,svc", "--all")
			err = deleteAll.Run()
			Expect(err).ShouldNot(HaveOccurred())
			Eventually(func() int {
				pc, _ := testHelper.PodCount("integration")
				return pc
			}, "20s").Should(Equal(0))

			deleteCM := exec.Command("kubectl", "delete", "configmap", "--all", "-n", "integration")
			err = deleteCM.Run()
			Expect(err).ShouldNot(HaveOccurred())
			Eventually(func() int {
				sc, _ := testHelper.ServiceCount("integration")
				return sc
			}, "20s").Should(Equal(0))
		})

		It("Returns a valid result", func() {
			var outputBytes []byte
			outputBytes, err = testHelper.RunCpi(rootTemplatePath, tmpConfigPath, agentPath, jsonPayload)
			Expect(err).ToNot(HaveOccurred())

			err = json.Unmarshal(outputBytes, &resultOutput)
			Expect(err).ToNot(HaveOccurred())
			Expect(resultOutput["result"]).ToNot(BeNil())
			Expect(resultOutput["error"]).To(BeNil())

			id := resultOutput["result"].(string)
			Expect(id).Should(ContainSubstring(clusterName))
			Expect(err).ToNot(HaveOccurred())
		})

		It("Creates the VM as a k8s pod", func() {
			_, err = testHelper.RunCpi(rootTemplatePath, tmpConfigPath, agentPath, jsonPayload)
			Expect(err).ToNot(HaveOccurred())

			Eventually(func() int {
				numberOfPods, err = testHelper.PodCount("integration")
				Expect(err).NotTo(HaveOccurred())
				return numberOfPods
			}, "30s").Should(Equal(1))
		})

		Context("When there are services in the cloud properties", func() {
			var numberOfServices int

			BeforeEach(func() {
				jsonPayload, err = testHelper.GenerateCpiJsonPayload("create_vm", rootTemplatePath, replacementMap)
				Expect(err).ToNot(HaveOccurred())

				numberOfServices, err = testHelper.ServiceCount("integration")
				Expect(err).NotTo(HaveOccurred())
				Expect(numberOfServices).To(Equal(0))
			})

			It("Creates the VM as a k8s pod", func() {
				_, err := testHelper.RunCpi(rootTemplatePath, tmpConfigPath, agentPath, jsonPayload)
				Expect(err).ToNot(HaveOccurred())

				Eventually(func() int {
					numberOfPods, err = testHelper.PodCount("integration")
					Expect(err).NotTo(HaveOccurred())
					return numberOfPods
				}, "30s").Should(Equal(1))
			})

			It("Creates the services with correct type and port", func() {
				_, err := testHelper.RunCpi(rootTemplatePath, tmpConfigPath, agentPath, jsonPayload)
				Expect(err).ToNot(HaveOccurred())

				Eventually(func() int {
					numberOfServices, err = testHelper.ServiceCount("integration")
					Expect(err).NotTo(HaveOccurred())
					return numberOfServices
				}, "30s").Should(Equal(7))

				directorService, err := testHelper.GetServiceByName("integration", "director1")
				Expect(err).NotTo(HaveOccurred())
				Expect(string(directorService.Spec.Type)).To(Equal("LoadBalancer"))
				Expect(int(directorService.Spec.Ports[0].NodePort)).To(Equal(32324))

				agentService, err := testHelper.GetServiceByName("integration", "agent1")
				Expect(err).NotTo(HaveOccurred())
				Expect(string(agentService.Spec.Type)).To(Equal("NodePort"))
				Expect(int(agentService.Spec.Ports[0].NodePort)).To(Equal(32323))

				blobstoreService, err := testHelper.GetServiceByName("integration", "blobstore1")
				Expect(err).NotTo(HaveOccurred())
				Expect(string(blobstoreService.Spec.Type)).To(Equal("ClusterIP"))

				haproxyService1, err := testHelper.GetServiceByName("integration", "ha-proxy-1")
				Expect(err).NotTo(HaveOccurred())
				Expect(string(haproxyService1.Spec.Type)).To(Equal("LoadBalancer"))
				Expect(int(haproxyService1.Spec.Ports[0].NodePort)).To(Equal(30080))
				Expect(haproxyService1.Spec.Selector).To(Equal(map[string]string{
					"bosh.cloudfoundry.org/job": "ha_proxy_z1",
				}))

				haproxyService2, err := testHelper.GetServiceByName("integration", "ha-proxy-2")
				Expect(err).NotTo(HaveOccurred())
				Expect(string(haproxyService2.Spec.Type)).To(Equal("NodePort"))
				Expect(int(haproxyService2.Spec.Ports[0].NodePort)).To(Equal(30443))
				Expect(haproxyService2.Spec.Selector).To(Equal(map[string]string{
					"bosh.cloudfoundry.org/job": "ha_proxy_z1",
				}))
				Expect(haproxyService2.Spec.ExternalIPs).To(Equal([]string{"158.10.10.10", "158.10.10.11"}))
			})
		})

		Context("When parameters are empty", func() {
			It("Returns an error", func() {
				jsonPayload := `{"method": "create_vm", "arguments": ["","",{},{},[],{}],"context": {}}`

				outputBytes, err := testHelper.RunCpi(rootTemplatePath, tmpConfigPath, agentPath, jsonPayload)
				Expect(err).ToNot(HaveOccurred())

				err = json.Unmarshal(outputBytes, &errorOutput)
				Expect(err).ToNot(HaveOccurred())
				Expect(errorOutput["result"]).To(Equal(""))
				Expect(errorOutput["error"]).ToNot(BeNil())
			})
		})
	})

	Context("Creating a VM (replicas)", func() {
		var numberOfPods int
		var numberOfReplicas int32
		var outputBytes []byte

		numReplicasInput := 2

		BeforeEach(func() {
			replacementMap["replicas"] = fmt.Sprintf("\"replicas\": %v,", numReplicasInput)

			jsonPayload, err = testHelper.GenerateCpiJsonPayload("create_vm", rootTemplatePath, replacementMap)
			Expect(err).ToNot(HaveOccurred())

			numberOfPods, err = testHelper.PodCount("integration")
			Expect(err).NotTo(HaveOccurred())
			Expect(numberOfPods).To(Equal(0))
		})

		AfterEach(func() {
			deleteAll := exec.Command("kubectl", "-n", "integration", "delete", "deploy,svc", "--all")
			err = deleteAll.Run()
			Expect(err).ShouldNot(HaveOccurred())
			Eventually(func() int {
				pc, _ := testHelper.PodCount("integration")
				return pc
			}, "20s").Should(Equal(0))

			deleteCM := exec.Command("kubectl", "delete", "configmap", "--all", "-n", "integration")
			err = deleteCM.Run()
			Expect(err).ShouldNot(HaveOccurred())
			Eventually(func() int {
				sc, _ := testHelper.ServiceCount("integration")
				return sc
			}, "20s").Should(Equal(0))
		})

		It("Returns a valid result", func() {
			var outputBytes []byte
			outputBytes, err = testHelper.RunCpi(rootTemplatePath, tmpConfigPath, agentPath, jsonPayload)
			Expect(err).ToNot(HaveOccurred())

			err = json.Unmarshal(outputBytes, &resultOutput)
			Expect(err).ToNot(HaveOccurred())
			Expect(resultOutput["result"]).ToNot(BeNil())
			Expect(resultOutput["error"]).To(BeNil())

			id := resultOutput["result"].(string)
			Expect(id).Should(ContainSubstring(clusterName))
			Expect(err).ToNot(HaveOccurred())
		})

		It("Creates the VM as a deployment with N replicas", func() {
			outputBytes, err = testHelper.RunCpi(rootTemplatePath, tmpConfigPath, agentPath, jsonPayload)
			Expect(err).ToNot(HaveOccurred())

			err = json.Unmarshal(outputBytes, &resultOutput)
			Expect(err).ToNot(HaveOccurred())
			Expect(resultOutput["result"]).ToNot(BeNil())
			Expect(resultOutput["error"]).To(BeNil())

			id := resultOutput["result"].(string)
			agentId := strings.TrimPrefix(id, clusterName+":")

			Eventually(func() int {
				numberOfPods, err = testHelper.PodCount("integration")
				Expect(err).NotTo(HaveOccurred())
				return numberOfPods
			}, "10s").Should(Equal(numReplicasInput))

			Eventually(func() int {
				numberOfReplicas, err = testHelper.ReplicaCount("integration", agentId)
				Expect(err).NotTo(HaveOccurred())
				return int(numberOfReplicas)
			}, "10s").Should(Equal(numReplicasInput))
		})
	})

	Describe("Creating an Ingress Service", func() {
		numReplicasInput := 1

		BeforeEach(func() {
			replacementMap["replicas"] = fmt.Sprintf("\"replicas\": %v,", numReplicasInput)
			jsonPayload, err = testHelper.GenerateCpiJsonPayload("create-ingress", rootTemplatePath, replacementMap)
			//fmt.Fprintln(os.Stderr, "PAYLOAD: ", jsonPayload)
			Expect(err).ToNot(HaveOccurred())

			numberOfServices, err := testHelper.ServiceCount("integration")
			Expect(err).NotTo(HaveOccurred())
			Expect(numberOfServices).To(Equal(0))

			numberOfIngresses, err := testHelper.IngressCount("integration")
			Expect(err).NotTo(HaveOccurred())
			Expect(numberOfIngresses).To(Equal(0))
		})

		AfterEach(func() {
			deleteAll := exec.Command("kubectl", "-n", "integration", "delete", "deploy,svc,ing", "--all")
			err = deleteAll.Run()
			Expect(err).ShouldNot(HaveOccurred())
			Eventually(func() int {
				pc, _ := testHelper.PodCount("integration")
				return pc
			}, "20s").Should(Equal(0))

			deleteCM := exec.Command("kubectl", "delete", "configmap", "--all", "-n", "integration")
			err = deleteCM.Run()
			Expect(err).ShouldNot(HaveOccurred())
			Eventually(func() int {
				sc, _ := testHelper.ServiceCount("integration")
				return sc
			}, "20s").Should(Equal(0))
		})

		It("Returns a valid result", func() {
			var outputBytes []byte
			outputBytes, err = testHelper.RunCpi(rootTemplatePath, tmpConfigPath, agentPath, jsonPayload)
			//fmt.Fprintln(os.Stderr, "OUTPUT: ", string(outputBytes))
			Expect(err).ToNot(HaveOccurred())

			err = json.Unmarshal(outputBytes, &resultOutput)
			Expect(err).ToNot(HaveOccurred())
			Expect(resultOutput["result"]).ToNot(BeNil())
			Expect(resultOutput["error"]).To(BeNil())

			id := resultOutput["result"].(string)
			Expect(id).Should(ContainSubstring(clusterName))
			Expect(err).ToNot(HaveOccurred())
		})

		It("Creates the services with correct type and port", func() {
			_, err := testHelper.RunCpi(rootTemplatePath, tmpConfigPath, agentPath, jsonPayload)
			Expect(err).ToNot(HaveOccurred())

			Eventually(func() int {
				numberOfServices, err := testHelper.ServiceCount("integration")
				Expect(err).NotTo(HaveOccurred())
				return numberOfServices
			}, "10s").Should(Equal(1))

			nginxService, err := testHelper.GetServiceByName("integration", "nginx")
			Expect(err).NotTo(HaveOccurred())
			//Expect(string(directorService.Spec.Type)).To(Equal("LoadBalancer"))
			Expect(int(nginxService.Spec.Ports[0].Port)).To(Equal(80))

			Eventually(func() int {
				numberOfIngresses, err := testHelper.IngressCount("integration")
				Expect(err).NotTo(HaveOccurred())
				return numberOfIngresses
			}, "40s").Should(Equal(2))

			ingress1Ingress, err := testHelper.GetIngressByName("integration", "ingress1")
			Expect(err).NotTo(HaveOccurred())
			Expect(ingress1Ingress.Spec.Backend.ServiceName).To(Equal("nginx"))
			Expect(ingress1Ingress.Spec.Backend.ServicePort).To(Equal(intstr.FromInt(80)))

			ingress2Ingress, err := testHelper.GetIngressByName("integration", "ingress2")
			Expect(err).NotTo(HaveOccurred())
			Expect(ingress2Ingress.Spec.TLS).To(ConsistOf(
				v1beta1.IngressTLS{
					Hosts:      []string{"apoorv-dev3.eu-central.containers.mybluemix.net"},
					SecretName: "apoorv-dev3",
				},
			))
			Expect(ingress2Ingress.Spec.Rules).To(ConsistOf(
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

			statusCode, err := testHelper.GetHTTPStatusCode("http://" + ingress2Ingress.Spec.Rules[0].Host + ingress2Ingress.Spec.Rules[0].IngressRuleValue.HTTP.Paths[0].Path)
			Expect(err).NotTo(HaveOccurred())
			Expect(statusCode).To(Equal(200))
		})

	})
})
