package disk

import (
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"k8s.io/client-go/pkg/api/v1"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	testHelper "github.ibm.com/Bluemix/kubernetes-cpi/integration/test_assets"
)

const agentPath = "integration/test_assets/cpi_methods/agent.json"
const configPath = "integration/test_assets/cpi_methods/config.json"

var _ = Describe("Disk and Volume Management", func() {
	var (
		err                             error
		jsonPayload                     string
		clusterName                     string
		kubeConfig                      string
		rootTemplatePath, tmpConfigPath string
		replacementMap                  map[string]string
		output                          map[string]interface{}
	)

	BeforeEach(func() {
		kubeConfig = os.Getenv("KUBECONFIG")
		Expect(err).ToNot(HaveOccurred())

		clusterName = os.Getenv("CLUSTER_NAME")
		Expect(err).ToNot(HaveOccurred())

		// This assumes you are in a certain directory - change?
		pwd, _ := os.Getwd()
		rootTemplatePath = filepath.Join(pwd, "..", "..")

		replacementMap = map[string]string{
			"context":            clusterName,
			"storageClass":       "ibmc-file-gold",
			"storageProvisioner": "ibm.io/ibmc-file",
		}

		tmpConfigPath, err = testHelper.CreateTmpConfigFile(rootTemplatePath, configPath, kubeConfig)
		Expect(err).ToNot(HaveOccurred())
	})

	AfterEach(func() {
		err = os.Remove(tmpConfigPath)
		Expect(err).ToNot(HaveOccurred())
	})

	Context("Creating a Disk", func() {
		var (
			pvcs v1.PersistentVolumeClaimList
		)

		BeforeEach(func() {
			jsonPayload, err = testHelper.GenerateCpiJsonPayload("create_disk", rootTemplatePath, replacementMap)
			Expect(err).ToNot(HaveOccurred())

			pvcs, err = testHelper.Pvcs("integration")
			Expect(err).NotTo(HaveOccurred())
			Expect(len(pvcs.Items)).To(Equal(0))
		})

		AfterEach(func() {
			deleteAll := exec.Command("kubectl", "-n", "integration", "delete", "pvc", "--all")
			err = deleteAll.Run()
			Expect(err).ShouldNot(HaveOccurred())

			Eventually(func() int {
				p, _ := testHelper.Pvcs("integration")
				return len(p.Items)
			}, "10s").Should(Equal(0))
		})

		It("Creates a disk of the correct size", func() {
			_, err := testHelper.RunCpi(rootTemplatePath, tmpConfigPath, agentPath, jsonPayload)
			Expect(err).ToNot(HaveOccurred())

			pvcs, err = testHelper.Pvcs("integration")
			Expect(err).NotTo(HaveOccurred())
			Expect(len(pvcs.Items)).To(Equal(1))

			capacity := pvcs.Items[0].Status.Capacity["storage"]
			Expect(capacity.String()).To(Equal("20Gi"))
		})
	})

	Context("Has a disk", func() {

		It("Does not have a disk when the disk does not exist", func() {
			replacementMap["diskID"] = "someJunkyDiskID"
			jsonPayload, err = testHelper.GenerateCpiJsonPayload("has_disk", rootTemplatePath, replacementMap)
			obytes, err := testHelper.RunCpi(rootTemplatePath, tmpConfigPath, agentPath, jsonPayload)

			Expect(err).ToNot(HaveOccurred())

			err = json.Unmarshal(obytes, &output)
			Expect(err).ToNot(HaveOccurred())

			result := output["result"].(bool)
			Expect(result).To(Equal(false))
		})

		It("Has a disk when the disk does exist", func() {
			// Create a disk
			jsonPayload, err = testHelper.GenerateCpiJsonPayload("create_disk", rootTemplatePath, replacementMap)
			Expect(err).ToNot(HaveOccurred())

			obytes, err := testHelper.RunCpi(rootTemplatePath, tmpConfigPath, agentPath, jsonPayload)
			Expect(err).ToNot(HaveOccurred())

			err = json.Unmarshal(obytes, &output)
			Expect(err).ToNot(HaveOccurred())

			diskID := strings.Split(output["result"].(string), ":")[1]
			replacementMap["diskID"] = diskID

			// run the has_disk cpi
			jsonPayload, err = testHelper.GenerateCpiJsonPayload("has_disk", rootTemplatePath, replacementMap)
			obytes, err = testHelper.RunCpi(rootTemplatePath, tmpConfigPath, agentPath, jsonPayload)
			Expect(err).ToNot(HaveOccurred())

			err = json.Unmarshal(obytes, &output)
			Expect(err).ToNot(HaveOccurred())

			result := output["result"].(bool)
			Expect(result).To(Equal(true))

			// delete the disk
			deleteAll := exec.Command("kubectl", "-n", "integration", "delete", "pvc", "--all")
			err = deleteAll.Run()
			Expect(err).ShouldNot(HaveOccurred())
		})
	})

	Context("Deleting a disk", func() {
		var pvcs v1.PersistentVolumeClaimList

		BeforeEach(func() {

			jsonPayload, err = testHelper.GenerateCpiJsonPayload("create_disk", rootTemplatePath, replacementMap)
			Expect(err).ToNot(HaveOccurred())

			obytes, err := testHelper.RunCpi(rootTemplatePath, tmpConfigPath, agentPath, jsonPayload)
			Expect(err).ToNot(HaveOccurred())

			err = json.Unmarshal(obytes, &output)
			Expect(err).ToNot(HaveOccurred())

			diskID := strings.Split(output["result"].(string), ":")[1]

			replacementMap["diskID"] = diskID

			jsonPayload, err = testHelper.GenerateCpiJsonPayload("delete_disk", rootTemplatePath, replacementMap)
			Expect(err).ToNot(HaveOccurred())

		})

		AfterEach(func() {
			deleteAll := exec.Command("kubectl", "-n", "integration", "delete", "pvc", "--all")
			err = deleteAll.Run()
			Expect(err).ShouldNot(HaveOccurred())

			Eventually(func() int {
				p, _ := testHelper.Pvcs("integration")
				return len(p.Items)
			}, "10s").Should(Equal(0))
		})

		It("Deletes the disk", func() {
			_, err = testHelper.RunCpi(rootTemplatePath, tmpConfigPath, agentPath, jsonPayload)
			Expect(err).ToNot(HaveOccurred())

			Eventually(func() int {
				pvcs, err = testHelper.Pvcs("integration")
				Expect(err).NotTo(HaveOccurred())

				pvcCount := len(pvcs.Items)
				return pvcCount
			}, "60s").Should(Equal(0))

		})

	})

})
