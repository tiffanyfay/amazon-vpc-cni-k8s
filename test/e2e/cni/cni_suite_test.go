package cni_test

import (
	"testing"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

// var _ = SynchronizedAfterSuite(func() {
// 	// Run on all Ginkgo nodes
// 	utils.Logf("Running AfterSuite actions on all nodes")
// 	framework.RunCleanupActions()
// }, func() {
// 	Expect(f).NotTo(BeNil())
// 	promResources.ExpectCleanupSuccessful(ctx, f, ns)
// 	testpodResources.ExpectDeploymentSuccessful(ctx, f, ns)
// })

func TestE2E(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "E2E Suite")
}
