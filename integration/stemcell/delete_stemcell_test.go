package stemcell_test

import (
	"encoding/json"
	"os"
	"path/filepath"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	testHelper "github.ibm.com/Bluemix/kubernetes-cpi/integration/test_assets"
	"fmt"
)

var _ = Describe("Deleting a Stemcell", func() {
	var (
		err                             error
		jsonPayload                     string
		kubeConfig                      string
		rootTemplatePath, tmpConfigPath string
		replacementMap                  map[string]string

		resultOutput                    map[string]interface{}
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

	Context("Deleting a Stemcell", func() {

		BeforeEach(func() {
			jsonPayload, err = testHelper.GenerateCpiJsonPayload("delete_stemcell", rootTemplatePath, replacementMap)
			Expect(err).ToNot(HaveOccurred())
		})

		It("Returns a valid result", func() {
			outputBytes, err := testHelper.RunCpi(rootTemplatePath, tmpConfigPath, agentPath, jsonPayload)
			Expect(err).ToNot(HaveOccurred())

			err = json.Unmarshal(outputBytes, &resultOutput)
			Expect(err).ToNot(HaveOccurred())
			Expect(resultOutput["result"]).To(BeNil())
			Expect(resultOutput["error"]).To(BeNil())
			Expect(resultOutput["log"]).To(BeEmpty())

		})
	})
})