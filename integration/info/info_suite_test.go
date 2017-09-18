package info_test

import (
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	testHelper "github.ibm.com/Bluemix/kubernetes-cpi/integration/test_assets"
	"testing"
)

func TestInfo(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Info Suite")
}

var _ = BeforeSuite(func() {
	err := testHelper.ConnectCluster()
	Expect(err).NotTo(HaveOccurred(), "Connecting cluster ...")
})
