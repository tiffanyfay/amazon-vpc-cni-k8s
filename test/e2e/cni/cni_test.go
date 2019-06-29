package cni_test

import (
	"context"
	"time"

	"github.com/aws/amazon-vpc-cni-k8s/test/e2e/framework"
	"github.com/aws/amazon-vpc-cni-k8s/test/e2e/framework/utils"
	"github.com/aws/amazon-vpc-cni-k8s/test/e2e/resources"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	promv1 "github.com/prometheus/client_golang/api/prometheus/v1"
	"github.com/prometheus/common/log"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// Timeout for waiting events in seconds
// const TIMEOUT = 60
var (
	f                *framework.Framework
	ns               *corev1.Namespace
	promResources    *resources.Resources
	testpodResources *resources.Resources
	prom             *resources.Prom
	promAPI          promv1.API
	err              error
	testTime         time.Time
	limit            float32
	ctx              context.Context
)

func SetUpPrometheus() {
	promReplicas := int32(1)
	limit = 0.1

	f, err = framework.NewFastFramework()
	Expect(err).NotTo(HaveOccurred())

	ns = &corev1.Namespace{ObjectMeta: metav1.ObjectMeta{Name: "cni-test"}}
	ctx = context.Background()

	// TODO see if this can be run on each node -- daemonset
	promResources = resources.NewPromResources(ns.Name, promReplicas)
	promResources.ExpectDeploymentSuccessful(ctx, f, ns)

	promAPI, err = resources.NewPromAPI(f, ns)
	Expect(err).NotTo(HaveOccurred()) // TODO: make sure this kills the test

	time.Sleep(time.Second * 5)

	prom = &resources.Prom{API: promAPI}

	// TODO should we make sure they are on each node?
	testpodResources := resources.NewTestpodResources(ns.Name, 6)
	testpodResources.ExpectDeploymentSuccessful(ctx, f, ns)
}

// TODO move to cni_suite_test.go
var _ = BeforeSuite(SetUpPrometheus)

var _ = Describe("Testing CNI E2E", func() {
	time.Sleep(time.Second * 5)
	testTime := time.Now()

	Context("With CNI testpods and prometheus metrics", func() {
		It("should get number of events received", func() {
			// TODO: set it for some # of expected requests?
			receivedTotal, err := prom.Query("cni_test_received_total", testTime)
			log.Infof("receivedTotal %v", receivedTotal)
			Expect(err).NotTo(HaveOccurred())
			Expect(receivedTotal).NotTo(BeNil())
		})
		It("should get dnsRequestFailurePercent below limit", func() {
			dnsRequestFailurePercent, err := prom.QueryPercent("cni_test_dns_request_total",
				"cni_test_dns_request_failure", testTime)
			Expect(err).NotTo(HaveOccurred())
			Expect(dnsRequestFailurePercent).NotTo(BeNil())
			log.Infof("dnsRequestFailurePercent %v", dnsRequestFailurePercent)
			Expect(dnsRequestFailurePercent).To(BeNumerically("<", limit))
		})
		It("should get externalHTTPRequestsFailurePercent below limit", func() {
			externalHTTPRequestsFailurePercent, err := prom.QueryPercent("cni_test_external_http_request_total",
				"cni_test_external_http_request_failure", testTime)
			Expect(err).NotTo(HaveOccurred())
			Expect(externalHTTPRequestsFailurePercent).NotTo(BeNil())
			Expect(externalHTTPRequestsFailurePercent).To(BeNumerically("<", limit))
		})
		It("should get svcClusterIPRequestFailurePercent below limit QueryPercent", func() {
			svcClusterIPRequestFailurePercent, err := prom.QueryPercent("cni_test_cluster_ip_request_total",
				"cni_test_cluster_ip_request_failure", testTime)
			Expect(err).NotTo(HaveOccurred())
			Expect(svcClusterIPRequestFailurePercent).NotTo(BeNil())
			Expect(svcClusterIPRequestFailurePercent).To(BeNumerically("<", limit))
		})
		It("should get svcPodIPRequestsFailurePercent below limit", func() {
			svcPodIPRequestsFailurePercent, err := prom.QueryPercent("cni_test_external_http_request_total",
				"cni_test_external_http_request_failure", testTime)
			Expect(err).NotTo(HaveOccurred())
			Expect(svcPodIPRequestsFailurePercent).NotTo(BeNil())
			Expect(svcPodIPRequestsFailurePercent).To(BeNumerically("<", limit))
		})
	})

	Context("With IPAMD and prometheus metrics", func() {
		It("awsCNIAWSAPIErrorCount should be 0", func() {
			QueryPercent, err := prom.Query("awscni_aws_api_error_count", testTime)
			Expect(err).NotTo(HaveOccurred())
			Expect(QueryPercent).NotTo(BeNil())
			Expect(QueryPercent).To(BeNumerically("<=", 5))
		})
	})

	Context("With default settings", func() {
		It("Should get 2 ENIs", func() {
			attachedENIs, err := f.AWSClient.GetAttachedENIs()
			Expect(err).ShouldNot(HaveOccurred())
			maxENIs, err := f.AWSClient.GetENILimit()
			Expect(err).ShouldNot(HaveOccurred())
			Expect(len(attachedENIs)).To(BeNumerically("<", maxENIs))
			Expect(len(attachedENIs)).To(Equal(2))
		})
	})

	Context("With ipamd allocating 3 ENIs with 2 ENIs full", func() {
		It("Should get 3 ENIs", func() {

			attachedENIs, err := f.AWSClient.GetAttachedENIs()
			Expect(err).ShouldNot(HaveOccurred())
			maxENIs, err := f.AWSClient.GetENILimit()
			Expect(err).ShouldNot(HaveOccurred())
			Expect(len(attachedENIs)).To(BeNumerically("<", maxENIs))
			Expect(len(attachedENIs)).To(Equal(2))
		})
	})

	// Context("With the metrics endpoint", func() {
	// 	Expect(f).ToNot(BeNil())
	// 	nodes, err := f.ClientSet.CoreV1().Nodes().List(metav1.ListOptions{})
	// 	Expect(err).ShouldNot(HaveOccurred())
	// 	for _, node := range nodes.Items {
	// 		It("Should get 2 ENIs", func() {
	// 			attachedENIs, err := f.AWSClient.GetAttachedENIs()
	// 			Expect(err).ShouldNot(HaveOccurred())
	// 			maxENIs, err := f.AWSClient.GetENILimit()
	// 			Expect(err).ShouldNot(HaveOccurred())
	// 			Expect(len(attachedENIs)).To(BeNumerically("<", maxENIs))
	// 			Expect(len(attachedENIs)).To(Equal(2))
	// 		})
	// 	}
	// })

	AfterEach(func() {
		// promResources.ExpectCleanupSuccessful(ctx, f, ns)
	})
})

var _ = SynchronizedAfterSuite(func() {
	// Run on all Ginkgo nodes
	utils.Logf("Running AfterSuite actions on all nodes")
	framework.RunCleanupActions()
}, func() {
	Expect(f).NotTo(BeNil())
	promResources.ExpectCleanupSuccessful(ctx, f, ns)
	testpodResources.ExpectDeploymentSuccessful(ctx, f, ns)
})
