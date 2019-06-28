package cni_test

import (
	"context"
	"fmt"
	"time"

	"github.com/aws/amazon-vpc-cni-k8s/test/e2e/framework"
	"github.com/aws/amazon-vpc-cni-k8s/test/e2e/resources"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	promv1 "github.com/prometheus/client_golang/api/prometheus/v1"
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
)

var _ = BeforeSuite(func() {
	var promReplicas int32 = 1
	limit = 0.1

	f, err = framework.NewFastFramework()
	if err != nil {
		panic(err.Error())
	}
	ns = &corev1.Namespace{ObjectMeta: metav1.ObjectMeta{Name: "cni-test"}}
	ctx := context.Background()

	promResources = resources.NewPromResources(ns.Name, promReplicas)
	promResources.ExpectDeploymentSuccessful(ctx, f, ns)

	promAPI, err = resources.NewPromAPI(f, ns)
	Expect(err).NotTo(HaveOccurred()) // TODO: make sure this kills the test

	time.Sleep(time.Second * 5)

	prom = &resources.Prom{
		API:      promAPI,
		TestTime: time.Now(),
	}

	testpodResources := resources.NewTestpodResources(ns.Name, 6)
	testpodResources.ExpectDeploymentSuccessful(ctx, f, ns)
})

var _ = Describe("cni-tester", func() {
	// var promReplicas int32 = 1
	// var limit float32 = 0.1

	// f, err = framework.NewFastFramework()
	// if err != nil {
	// 	panic(err.Error())
	// }
	// ns = &corev1.Namespace{ObjectMeta: metav1.ObjectMeta{Name: "cni-test"}}
	// ctx := context.Background()

	// promResources = resources.NewPromResources(ns.Name, promReplicas)
	// promResources.ExpectDeploymentSuccessful(ctx, f, ns)

	// time.Sleep(time.Second * 5)

	// promAPI, err = resources.NewPromAPI(f, ns)
	// Expect(err).NotTo(HaveOccurred()) // TODO: make sure this kills the test

	// time.Sleep(time.Second * 5)

	// prom = &resources.Prom{
	// 	API:      promAPI,
	// 	TestTime: time.Now(),
	// }

	// testpodResources := resources.NewTestpodResources(ns.Name, 6)
	// testpodResources.ExpectDeploymentSuccessful(ctx, f, ns)

	It("should get number of events received", func() {
		// TODO: set it for some # of expected requests?
		received, err := prom.Query("cni_test_received_total")
		Expect(err).NotTo(HaveOccurred())
		Expect(received).NotTo(BeNil())
	})

	It("should get dnsRequestFailurePercent below limit", func() {
		dnsRequestFailurePercent, err := prom.QueryPercent("cni_test_external_http_request_total", "cni_test_dns_request_failure")
		Expect(err).NotTo(HaveOccurred())
		Expect(dnsRequestFailurePercent).NotTo(BeNil())
		Expect(dnsRequestFailurePercent).To(BeNumerically("<", limit))
	})

	It("should get externalHTTPRequestsFailurePercent below limit", func() {
		externalHTTPRequestsFailurePercent, err := prom.QueryPercent("cni_test_external_http_request_total", "cni_test_external_http_request_failure")
		Expect(err).NotTo(HaveOccurred())
		Expect(externalHTTPRequestsFailurePercent).NotTo(BeNil())
		Expect(externalHTTPRequestsFailurePercent).To(BeNumerically("<", limit))
	})

	It("should get svcClusterIPRequestFailurePercent below limit QueryPercent", func() {
		svcClusterIPRequestFailurePercent, err := prom.QueryPercent("cni_test_cluster_ip_request_total", "cni_test_cluster_ip_request_failure")
		Expect(err).NotTo(HaveOccurred())
		Expect(svcClusterIPRequestFailurePercent).NotTo(BeNil())
		Expect(svcClusterIPRequestFailurePercent).To(BeNumerically("<", limit))
	})

	It("should get svcPodIPRequestsFailurePercent below limit", func() {
		svcPodIPRequestsFailurePercent, err := prom.QueryPercent("cni_test_external_http_request_total", "cni_test_external_http_request_failure")
		fmt.Println(svcPodIPRequestsFailurePercent)
		Expect(err).NotTo(HaveOccurred())
		Expect(svcPodIPRequestsFailurePercent).NotTo(BeNil())
		Expect(svcPodIPRequestsFailurePercent).To(BeNumerically("<", limit))
	})

	It("awsCNIAWSAPIErrorCount should be 0", func() {
		QueryPercent, err := prom.Query("awscni_aws_api_error_count")
		Expect(err).NotTo(HaveOccurred())
		Expect(QueryPercent).NotTo(BeNil())
		Expect(QueryPercent).To(BeNumerically("<=", 5))
	})

	It("Should get 2 ENIs", func() {
		attachedENIs, err := f.AWSClient.GetAttachedENIs()
		Expect(err).ShouldNot(HaveOccurred())
		maxENIs, err := f.AWSClient.GetENILimit()
		Expect(err).ShouldNot(HaveOccurred())
		Expect(len(attachedENIs)).To(BeNumerically("<", maxENIs))
		Expect(len(attachedENIs)).To(Equal(2))
	})

	// It("Should get 2 ENIs", func() {
	// 	attachedENIs, err := f.AWSClient.GetAttachedENIs()
	// 	Expect(err).ShouldNot(HaveOccurred())
	// 	maxENIs, err := f.AWSClient.GetENILimit()
	// 	Expect(err).ShouldNot(HaveOccurred())
	// 	Expect(len(attachedENIs)).To(BeNumerically("<", maxENIs))
	// 	Expect(len(attachedENIs)).To(Equal(2))
	// })

	//promResources.ExpectCleanupSuccessful(ctx, f, ns)
	//testpodResources.ExpectDeploymentSuccessful(ctx, f, ns)

	AfterEach(func() {
		// promResources.ExpectCleanupSuccessful(ctx, f, ns)
	})
})
