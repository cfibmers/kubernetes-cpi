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

var _ = Describe("Integration test for create_vm", func() {
	var (
		err                             error
		clusterName                     string
		kubeConfig                      string
		rootTemplatePath, tmpConfigPath string
		replacementMap                  map[string]string
		errorOutput                     map[string]interface{}
		resultOutput                    map[string]interface{}
	)

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

		tmpConfigPath, err = testHelper.CreateTmpConfigPath(rootTemplatePath, configPath, kubeConfig)
		Expect(err).ToNot(HaveOccurred())
	})

	AfterEach(func() {
		err = os.RemoveAll(tmpConfigPath)
		Expect(err).ToNot(HaveOccurred())
	})

	Context("create_vm in cluster", func() {
		It("returns error because empty parameters", func() {
			jsonPayload := `{"method": "create_vm", "arguments": ["","",{},{},[],{}],"context": {}}`

			outputBytes, err := testHelper.RunCpi(rootTemplatePath, tmpConfigPath, agentPath, jsonPayload)
			Expect(err).ToNot(HaveOccurred())

			err = json.Unmarshal(outputBytes, &errorOutput)
			Expect(err).ToNot(HaveOccurred())
			Expect(errorOutput["result"]).To(Equal(""))
			Expect(errorOutput["error"]).ToNot(BeNil())
		})

		It("returns valid result because valid parameters", func() {
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

			deleteAll := exec.Command("kubectl", "-n", "integration", "delete", "po,svc", "--all")
			err = deleteAll.Run()
			Expect(err).ShouldNot(HaveOccurred())

			deleteCM := exec.Command("kubectl", "delete", "configmap", "agent-0fd9ff80-d39c-47da-6827-e1825bc8a999", "-n", "integration")
			err = deleteCM.Run()
			Expect(err).ShouldNot(HaveOccurred())
		})
	})
})
