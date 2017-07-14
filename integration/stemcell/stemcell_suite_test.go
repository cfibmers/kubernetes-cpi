package stemcell_test

import (
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	testHelper "github.ibm.com/Bluemix/kubernetes-cpi/integration/test_assets"
	"testing"
)

func TestStemcell(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Stemcell Suite")
}

var _ = BeforeSuite(func() {
	err := testHelper.ConnectCluster()
	Expect(err).NotTo(HaveOccurred(), "Connecting cluster ...")

	testHelper.CreateNamespace("integration")
})

var _ = AfterSuite(func() {
	testHelper.DeleteNamespace("integration")
})
