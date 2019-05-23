package e2e_test

import (
	"testing"

	_ "github.com/aws/amazon-vpc-cni-k8s/test/e2e/cni"
	"github.com/aws/amazon-vpc-cni-k8s/test/e2e/framework"
	"github.com/aws/amazon-vpc-cni-k8s/test/e2e/framework/utils"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = SynchronizedAfterSuite(func() {
	// Run on all Ginkgo nodes
	utils.Logf("Running AfterSuite actions on all nodes")
	framework.RunCleanupActions()
}, func() {
})

func TestE2e(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "E2E Suite")
}
