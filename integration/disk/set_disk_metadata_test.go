package disk

import (
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	testHelper "github.ibm.com/Bluemix/kubernetes-cpi/integration/test_assets"
	"k8s.io/client-go/pkg/api/v1"
	"fmt"
)

var _ = Describe("Set Disk Metadata", func() {
	var (
		err                             error
		jsonPayload                     string
		clusterName                     string
		kubeConfig                      string
		rootTemplatePath, tmpConfigPath string
		replacementMap                  map[string]string
		resultOutput                    map[string]interface{}
		pvcs                            v1.PersistentVolumeClaimList
		diskID                          string
		agentId                         string
		pvcName                         string
		pvc                             v1.PersistentVolumeClaim
	)

	CreateDisk := func() {
		replacementMap = map[string]string{
			"context": clusterName,
		}

		jsonPayload, err = testHelper.GenerateCpiJsonPayload("create_disk", rootTemplatePath, replacementMap)
		Expect(err).ToNot(HaveOccurred())

		pvcs, err = testHelper.Pvcs("integration")
		Expect(err).NotTo(HaveOccurred())
		Expect(len(pvcs.Items)).To(Equal(0))

		obytes, err := testHelper.RunCpi(rootTemplatePath, tmpConfigPath, agentPath, jsonPayload)
		Expect(err).ToNot(HaveOccurred())

		err = json.Unmarshal(obytes, &resultOutput)
		Expect(err).ToNot(HaveOccurred())

		diskID = strings.Split(resultOutput["result"].(string), ":")[1]

		pvcs, err = testHelper.Pvcs("integration")
		Expect(err).NotTo(HaveOccurred())
		Expect(len(pvcs.Items)).To(Equal(1))
	}

	BeforeEach(func() {
		kubeConfig = os.Getenv("KUBECONFIG")
		Expect(err).ToNot(HaveOccurred())

		clusterName = os.Getenv("CLUSTER_NAME")
		Expect(err).ToNot(HaveOccurred())

		pwd, _ := os.Getwd()
		rootTemplatePath = filepath.Join(pwd, "..", "..")

		replacementMap = map[string]string{
			"context": clusterName,
		}

		tmpConfigPath, err = testHelper.CreateTmpConfigFile(rootTemplatePath, configPath, kubeConfig)
		Expect(err).ToNot(HaveOccurred())
	})

	AfterEach(func() {
		Expect(err).ToNot(HaveOccurred())
	})

	Context("Set Disk Metadata", func() {
		BeforeEach(func() {
			CreateDisk()
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

		It("Set Disk Metadata successfully", func() {
			replacementMap = map[string]string{
				"context": clusterName,
				"diskID":  diskID,
				"agentID": agentId,
			}

			pvcName = fmt.Sprintf("disk-%s", diskID)
			pvc, err = testHelper.GetPvcByName(pvcName, "integration")
			Expect(err).ToNot(HaveOccurred())

			Expect(len(pvc.ObjectMeta.Labels)).To(Equal(1))

			jsonPayload, err := testHelper.GenerateCpiJsonPayload("set_disk_metadata", rootTemplatePath, replacementMap)
			Expect(err).ToNot(HaveOccurred())

			outputBytes, err := testHelper.RunCpi(rootTemplatePath, tmpConfigPath, agentPath, jsonPayload)
			Expect(err).ToNot(HaveOccurred())

			err = json.Unmarshal(outputBytes, &resultOutput)
			Expect(err).ToNot(HaveOccurred())
			Expect(resultOutput["result"]).To(BeNil())
			Expect(resultOutput["error"]).To(BeNil())

			pvc, err = testHelper.GetPvcByName(pvcName, "integration")
			Expect(len(pvc.ObjectMeta.Labels)).To(Equal(6))
			Expect(pvc.ObjectMeta.Labels["bosh.cloudfoundry.org/director"]).To(Equal("bosh"))
			Expect(pvc.ObjectMeta.Labels["bosh.cloudfoundry.org/deployment"]).To(Equal("cf-kube"))
			Expect(pvc.ObjectMeta.Labels["bosh.cloudfoundry.org/agent-id"]).To(Equal(agentId))
			Expect(pvc.ObjectMeta.Labels["bosh.cloudfoundry.org/instance_index"]).To(Equal("0"))
			Expect(pvc.ObjectMeta.Labels["bosh.cloudfoundry.org/attached_at"]).To(Equal("2017-08-17T03_51_15Z"))
		})
	})
})
