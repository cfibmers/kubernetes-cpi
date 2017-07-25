package vm_test

import (
	"testing"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	testHelper "github.ibm.com/Bluemix/kubernetes-cpi/integration/test_assets"
)

func TestVM(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "VM Suite")
}

var _ = BeforeSuite(func() {
	err := testHelper.ConnectCluster()
	Expect(err).NotTo(HaveOccurred(), "Connecting cluster ...")
})

var _ = AfterSuite(func() {
	testHelper.DeleteNamespace("integration")
})
