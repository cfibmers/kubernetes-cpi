package create_vm_test

import (
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	testHelper "github.ibm.com/Bluemix/kubernetes-cpi/integration/test_assets"
	"testing"
)

func TestCreateVm(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "CreateVm Suite")
}

var _ = BeforeSuite(func() {
	_ = testHelper.ConnectCluster()

})
