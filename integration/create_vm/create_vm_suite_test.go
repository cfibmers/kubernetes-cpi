package create_vm_test

import (
	"testing"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	testHelper "github.ibm.com/Bluemix/kubernetes-cpi/integration/test_assets"
)

func TestCreateVM(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "CreateVM Suite")
}

var _ = BeforeSuite(func() {
	err := testHelper.ConnectCluster()
	Expect(err).NotTo(HaveOccurred(), "Connecting cluster ...")
})

var _ = AfterSuite(func() {
	testHelper.DeleteNamespace("integration")
})
