package vm_test

import (
	"encoding/json"
	"os"
	"path/filepath"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	testHelper "github.ibm.com/Bluemix/kubernetes-cpi/integration/test_assets"
)

var _ = Describe("Integration test for vm", func() {
	var (
		clusterName                     string
		kubeConfig                      string
		rootTemplatePath, tmpConfigPath string
		replacementMap                  map[string]string
		resultOutput                    map[string]interface{}
		err                             error
	)

	CreateVM := func() {
		var numberOfPods int
		var numberOfServices int

		jsonPayload, err := testHelper.GenerateCpiJsonPayload("create_vm", rootTemplatePath, replacementMap)
		Expect(err).ToNot(HaveOccurred())

		outputBytes, err := testHelper.RunCpi(rootTemplatePath, tmpConfigPath, agentPath, jsonPayload)
		Expect(err).ToNot(HaveOccurred())

		err = json.Unmarshal(outputBytes, &resultOutput)
		Expect(err).ToNot(HaveOccurred())
		Expect(resultOutput["result"]).ToNot(BeNil())
		Expect(resultOutput["error"]).To(BeNil())

		id := resultOutput["result"].(string)
		Expect(id).Should(ContainSubstring(clusterName))
		Expect(err).ToNot(HaveOccurred())

		numberOfPods, err = testHelper.PodCount("integration")
		Expect(err).NotTo(HaveOccurred())
		Expect(numberOfPods).To(Equal(1))

		numberOfServices, err = testHelper.ServiceCount("integration")
		Expect(err).NotTo(HaveOccurred())
		Expect(numberOfServices).To(Equal(5))
	}

	BeforeEach(func() {
		clusterName = os.Getenv("CLUSTER_NAME")
		Expect(err).ToNot(HaveOccurred())

		kubeConfig = os.Getenv("KUBECONFIG")
		Expect(err).ToNot(HaveOccurred())

		pwd, err := os.Getwd()
		Expect(err).ToNot(HaveOccurred())
		rootTemplatePath = filepath.Join(pwd, "..", "..")

		replacementMap = map[string]string{
			"context": clusterName,
		}

		tmpConfigPath, err = testHelper.CreateTmpConfigFile(rootTemplatePath, configPath, kubeConfig)
		Expect(err).ToNot(HaveOccurred())
	})

	Context("delete_vm with a valid VM ID", func() {
		var numberOfPods int
		var numberOfServices int

		BeforeEach(func() {
			CreateVM()
		})

		It("deletes the VM successfully", func() {
			jsonPayload, err := testHelper.GenerateCpiJsonPayload("delete_vm", rootTemplatePath, replacementMap)
			Expect(err).ToNot(HaveOccurred())

			outputBytes, err := testHelper.RunCpi(rootTemplatePath, tmpConfigPath, agentPath, jsonPayload)
			Expect(err).ToNot(HaveOccurred())

			err = json.Unmarshal(outputBytes, &resultOutput)
			Expect(err).ToNot(HaveOccurred())
			Expect(resultOutput["result"]).To(BeNil())
			Expect(resultOutput["error"]).To(BeNil())

			numberOfPods, err = testHelper.PodCount("integration")
			Expect(err).NotTo(HaveOccurred())
			Eventually(numberOfPods, "10s").Should(Equal(0))

			numberOfServices, err = testHelper.ServiceCount("integration")
			Expect(err).NotTo(HaveOccurred())
			Eventually(numberOfServices, "10s").Should(Equal(0))
		})

	})
})
