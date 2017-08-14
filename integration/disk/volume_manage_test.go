package disk

import (
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"fmt"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	testHelper "github.ibm.com/Bluemix/kubernetes-cpi/integration/test_assets"
	"k8s.io/client-go/pkg/api/v1"
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
		resultOutput                    map[string]interface{}
		numberOfPods                    int
		pvcs                            v1.PersistentVolumeClaimList
		diskID                          string
		oriPod, newPod                  v1.Pod
		agentId                         string
		podName                         string
	)

	CreateVM := func() {
		agentId = "490c18a5-3bb4-4b92-8550-ee4a1e955624"
		replacementMap = map[string]string{
			"agentID": agentId,
			"context": clusterName,
		}

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
	}

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
		err = os.Remove(tmpConfigPath)
		Expect(err).ToNot(HaveOccurred())
	})

	Context("Attaching a Disk", func() {
		BeforeEach(func() {
			CreateVM()
			CreateDisk()

			podName = fmt.Sprintf("agent-%s", agentId)
			oriPod, err = testHelper.GetPodByName(podName, "integration")
			Expect(err).ToNot(HaveOccurred())
			Expect(testHelper.PodCount("integration")).To(Equal(1))

			pvcs, err = testHelper.Pvcs("integration")
			Expect(err).ToNot(HaveOccurred())
			Expect(len(pvcs.Items)).To(Equal(1))

			replacementMap = map[string]string{
				"context": clusterName,
				"diskID":  diskID,
				"agentID": agentId,
			}

			jsonPayload, err = testHelper.GenerateCpiJsonPayload("attach_disk", rootTemplatePath, replacementMap)
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

		It("Attach a valid disk to a pod", func() {
			_, err := testHelper.RunCpi(rootTemplatePath, tmpConfigPath, agentPath, jsonPayload)
			Expect(err).ToNot(HaveOccurred())

			pvcs, err = testHelper.Pvcs("integration")
			Expect(err).NotTo(HaveOccurred())
			Expect(len(pvcs.Items)).To(Equal(1))
			Expect(testHelper.PodCount("integration")).To(Equal(1))

			newPod, err = testHelper.GetPodByName(podName, "integration")
			Eventually(len(newPod.Spec.Volumes), 600).Should(BeNumerically(">", len(oriPod.Spec.Volumes)))
			Expect(newPod.Spec.Hostname).To(Equal(oriPod.Spec.Hostname))
		})
	})
})
