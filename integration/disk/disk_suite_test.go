package disk_test

import (
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	"testing"

	testHelper "github.ibm.com/Bluemix/kubernetes-cpi/integration/test_assets"
)

func TestDisk(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "CreateDisk Suite")
}

var _ = BeforeSuite(func() {
	err := testHelper.ConnectCluster()
	Expect(err).NotTo(HaveOccurred(), "Connecting cluster ...")

	testHelper.CreateNamespace("integration")
})

var _ = AfterSuite(func() {
	testHelper.DeleteNamespace("integration")
})
