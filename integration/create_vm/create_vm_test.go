package create_vm_test

import (
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	testHelper "github.ibm.com/Bluemix/kubernetes-cpi/integration/test_assets"
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
	)

	BeforeEach(func() {
		kubeConfig = os.Getenv("KUBECONFIG")
		Expect(err).ToNot(HaveOccurred())

		// This assumes you are in a certain directory - change?
		pwd, _ := os.Getwd()
		rootTemplatePath = filepath.Join(pwd, "..", "..")

		replacementMap = map[string]string{
			"context": clusterName,
		}

		tmpConfigPath, err = testHelper.CreateTmpConfigFile(rootTemplatePath, configPath, kubeConfig)
		Expect(err).ToNot(HaveOccurred())
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
			}, "10s").Should(Equal(0))

			deleteCM := exec.Command("kubectl", "delete", "configmap", "--all", "-n", "integration")
			err = deleteCM.Run()
			Expect(err).ShouldNot(HaveOccurred())
			Eventually(func() int {
				sc, _ := testHelper.ServiceCount("integration")
				return sc
			}, "10s").Should(Equal(0))
		})

		It("Returns a valid result", func() {
			outputBytes, err := testHelper.RunCpi(rootTemplatePath, tmpConfigPath, agentPath, jsonPayload)
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
			_, err := testHelper.RunCpi(rootTemplatePath, tmpConfigPath, agentPath, jsonPayload)
			Expect(err).ToNot(HaveOccurred())

			numberOfPods, err = testHelper.PodCount("integration")
			Expect(err).NotTo(HaveOccurred())
			Expect(numberOfPods).To(Equal(1))
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

				numberOfPods, err = testHelper.PodCount("integration")
				Expect(err).NotTo(HaveOccurred())
				Expect(numberOfPods).To(Equal(1))
			})

			It("Creates the services", func() {
				_, err := testHelper.RunCpi(rootTemplatePath, tmpConfigPath, agentPath, jsonPayload)
				Expect(err).ToNot(HaveOccurred())

				numberOfServices, err = testHelper.ServiceCount("integration")
				Expect(err).NotTo(HaveOccurred())
				Expect(numberOfServices).To(Equal(5))
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
})
