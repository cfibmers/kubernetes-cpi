package actions_test

import (
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.ibm.com/Bluemix/kubernetes-cpi/actions"
)

var _ = Describe("Info", func() {
	Describe("Info", func() {
		It("returns the info about CPI", func() {
			info := actions.Info()
			expect_info := make(map[string]string)
			expect_info["api_version"] = "1.0"
			Expect(info).To(Equal(expect_info))
		})
	})
})
