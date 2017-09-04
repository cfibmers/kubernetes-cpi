package info

import (
	"encoding/json"
	"os"
	"path/filepath"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	testHelper "github.ibm.com/Bluemix/kubernetes-cpi/integration/test_assets"
)

const configPath = "integration/test_assets/cpi_methods/config.json"
const agentPath = "integration/test_assets/cpi_methods/agent.json"

var _ = Describe("Checking info", func() {
	var (
		err                             error
		jsonPayload                     string
		kubeConfig                      string
		rootTemplatePath, tmpConfigPath string
		replacementMap                  map[string]string

		resultOutput map[string]interface{}
	)

	BeforeEach(func() {
		kubeConfig = os.Getenv("KUBECONFIG")
		Expect(err).ToNot(HaveOccurred())

		// This assumes you are in a certain directory - change?
		pwd, _ := os.Getwd()
		rootTemplatePath = filepath.Join(pwd, "..", "..")

		tmpConfigPath, err = testHelper.CreateTmpConfigFile(rootTemplatePath, configPath, kubeConfig)
		Expect(err).ToNot(HaveOccurred())
	})

	AfterEach(func() {
		err = os.Remove(tmpConfigPath)
		Expect(err).ToNot(HaveOccurred())
	})

	Context("Checking info", func() {

		BeforeEach(func() {
			jsonPayload, err = testHelper.GenerateCpiJsonPayload("info", rootTemplatePath, replacementMap)
			Expect(err).ToNot(HaveOccurred())
		})

		It("Returns a valid result", func() {
			outputBytes, err := testHelper.RunCpi(rootTemplatePath, tmpConfigPath, agentPath, jsonPayload)
			Expect(err).ToNot(HaveOccurred())
			expect_info := make(map[string]interface{})
			expect_info["api_version"] = "1.0"
			err = json.Unmarshal(outputBytes, &resultOutput)
			Expect(err).ToNot(HaveOccurred())
			Expect(resultOutput["result"]).ToNot(BeNil())
			Expect(resultOutput["error"]).To(BeNil())
			Expect(resultOutput["log"]).To(BeEmpty())
			Expect(resultOutput["result"]).To(Equal(expect_info))
		})
	})
})
