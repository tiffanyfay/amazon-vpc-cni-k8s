package cni_test

import (
	"context"
	"fmt"
	"time"

	"github.com/aws/amazon-vpc-cni-k8s/test/e2e/cni"
	"github.com/aws/amazon-vpc-cni-k8s/test/e2e/framework"
	"github.com/aws/amazon-vpc-cni-k8s/test/e2e/framework/utils"
	"github.com/aws/amazon-vpc-cni-k8s/test/e2e/resources"

	log "github.com/cihub/seelog"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	promv1 "github.com/prometheus/client_golang/api/prometheus/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// Timeout for waiting events in seconds// const TIMEOUT = 60
var (
	err      error
	limit    float32
	ctx      context.Context
	testTime time.Time

	f                *framework.Framework
	prom             *resources.Prom
	awsNodeSvc       *resources.Resources
	promResources    *resources.Resources
	testpodResources *resources.Resources
	resourcesGroup   []*resources.Resources

	promAPI promv1.API
	ns      *corev1.Namespace
	// nodes   []corev1.Node
)

// func setup() {
// 	limit = 0.1

// 	f, err = framework.NewFastFramework()
// 	Expect(err).NotTo(HaveOccurred())

// 	ns = &corev1.Namespace{ObjectMeta: metav1.ObjectMeta{Name: "cni-test"}}
// 	ctx = context.Background()

// 	// TODO should we make sure they are on each node?
// 	// testpodResources = resources.NewTestpodResources(ns.Name, 6)
// 	// testpodResources.ExpectDeploySuccessful(ctx, f, ns)
// }

func expectTestpodPromMetricsPass() {
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
}

func expectAWSNodePromMetricsPass() {

}

// TODO move to cni_suite_test.go
// var _ = BeforeSuite(setup)
func createNginxResources(nodes []corev1.Node) []*resources.Resources {
	// Create NGINX resources
	// testpodResources.ExpectDeploymentScaleSuccessful(ctx, f, ns, 23)
	// Create NGINX resources
	var resourcesGroup []*resources.Resources
	Expect(len(nodes)).To(BeNumerically(">", 0))
	for i, node := range nodes {
		log.Infof("Creating deployment for node %d/%d: %v", i+1, len(nodes), node.Name)
		resource := resources.NewNginxResources(ns.Name, node.Name, 0)
		resource.ExpectDeploySuccessful(ctx, f, ns)
		resourcesGroup = append(resourcesGroup, resource)
	}
	return resourcesGroup
}

var _ = Describe("Testing CNI", func() {
	f = framework.New()

	ns = &corev1.Namespace{ObjectMeta: metav1.ObjectMeta{Name: "cni-test"}}
	ctx = context.Background() // TODO make this have a timeout

	kubeSystem := &corev1.Namespace{ObjectMeta: metav1.ObjectMeta{Name: "kube-system"}}

	BeforeEach(func() {
		// Create prometheus resources
		// testerNode, err := getTesterNodeName()
		// promResources = resources.NewPromResources(ns.Name, testerNode, 1)
		// promResources.ExpectDeploySuccessful(ctx, f, ns)
		// promAPI, err = resources.NewPromAPI(f, ns)
		// Expect(err).NotTo(HaveOccurred())
		// time.Sleep(time.Second * 5)
		// prom = &resources.Prom{API: promAPI}
		// time.Sleep(time.Second * 5)

		// 	ns = &corev1.Namespace{ObjectMeta: metav1.ObjectMeta{Name: "cni-test"}}
		// 	ctx = context.Background()

		// 	// TODO should we make sure they are on each node?

	})

	// Context("With CNI testpods and prometheus metrics", expectTestpodPromMetricsPass)

	// Context("With IPAMD and prometheus metrics", func() {
	// 	It("awsCNIAWSAPIErrorCount should be 0", func() {
	// 		query, err := prom.Query("awscni_aws_api_error_count", testTime)
	// 		Expect(err).NotTo(HaveOccurred())
	// 		Expect(query).NotTo(BeNil())
	// 		Expect(query).To(BeNumerically("<=", 5))
	// 	})
	// 	It("awscni_ipamd_error_count should be 0", func() {
	// 		QueryPercent, err := prom.Query("awscni_ipamd_error_count", testTime)
	// 		Expect(err).NotTo(HaveOccurred())
	// 		Expect(QueryPercent).NotTo(BeNil())
	// 		Expect(QueryPercent).To(BeNumerically("<=", 5))
	// 	})
	// 	// TODO query for each instance
	// 	// It("awscni_eni_max should be 4", func() {
	// 	// 	query, err := prom.Query("awscni_eni_max", testTime)
	// 	// 	Expect(err).NotTo(HaveOccurred())
	// 	// 	Expect(query).NotTo(BeNil())
	// 	// 	Expect(query).To(Equal(4))
	// 	// })
	// 	// It("awscni_ip_max should be 15", func() {
	// 	// 	query, err := prom.Query("awscni_ip_max", testTime)
	// 	// 	Expect(err).NotTo(HaveOccurred())
	// 	// 	Expect(query).NotTo(BeNil())
	// 	// 	Expect(query).To(Equal(15))
	// 	// })
	// 	// It("awsCNIAWSAPIErrorCount should be 0", func() {
	// 	// 	QueryPercent, err := prom.Query("awscni_aws_api_error_count", testTime)
	// 	// 	Expect(err).NotTo(HaveOccurred())
	// 	// 	Expect(QueryPercent).NotTo(BeNil())
	// 	// 	Expect(QueryPercent).To(BeNumerically("<=", 5))
	// 	// })
	// })

	It("Should pass with WARM_IP_TARGET=0 (default), WARM_ENI_TARGET=1 (default), and MAX_ENI=-1(default)", func() {
		cni.ExpectUpdateAWSNodeSuccessful(ctx, f, "0", "1", "-1")

		initialNodes, err := cni.GetTestNodes(f)
		Expect(err).ShouldNot(HaveOccurred())

		err = cni.ReplaceASGInstances(ctx, f, initialNodes)
		Expect(err).ShouldNot(HaveOccurred())

		awsNodeDS, err := f.ClientSet.AppsV1().DaemonSets(kubeSystem.Name).Get("aws-node", metav1.GetOptions{})
		Expect(err).ShouldNot(HaveOccurred())
		f.ResourceManager.WaitDaemonSetReady(ctx, awsNodeDS)

		nodes, err := cni.GetTestNodes(f)
		Expect(err).ShouldNot(HaveOccurred())
		resourcesGroup = createNginxResources(nodes)

		// TODO handle checking if coreDNS is on the instance
		for i, node := range nodes {
			By(fmt.Sprintf("scaling up pods in deployment %s to get 3 ENIs", resourcesGroup[i].Deployment.Name))
			_, ipLimit, err := cni.GetInstanceLimits(f, node.Name)
			Expect(err).ToNot(HaveOccurred())
			resourcesGroup[i].ExpectDeploymentScaleSuccessful(ctx, f, ns, int32(ipLimit*2))

			time.Sleep(time.Second * 5)
			cni.TestENIInfo(ctx, f, node, 3, ipLimit)

			By(fmt.Sprintf("scaling up pods in deployment %s to get 4 ENIs", resourcesGroup[i].Deployment.Name))
			resourcesGroup[i].ExpectDeploymentScaleSuccessful(ctx, f, ns, int32(ipLimit*2+1))

			time.Sleep(time.Second * 15) // TODO handle this because the ENI info is slower to load
			cni.TestENIInfo(ctx, f, node, 4, ipLimit)
		}
	})

	// It("Should pass with WARM_IP_TARGET=0 (default), WARM_ENI_TARGET=0, and MAX_ENI=-1(default)", func() {
	// 	expectUpdateAWSNodeSuccessful("0", "0", "-1")
	// 	nodes, err := getTestNodes()
	// 	Expect(err).ShouldNot(HaveOccurred())
	// 	err = replaceASGInstances(nodes)
	// 	Expect(err).ShouldNot(HaveOccurred())
	// 	awsNodeDS, err := f.ClientSet.AppsV1().DaemonSets(kubeSystem.Name).Get("aws-node", metav1.GetOptions{})
	// 	Expect(err).ShouldNot(HaveOccurred())
	// 	awsNodeS, err := f.ClientSet.CoreV1().Services(kubeSystem.Name).Get("aws-node", metav1.GetOptions{})
	// 	Expect(err).ShouldNot(HaveOccurred())
	// 	f.ResourceManager.WaitDaemonSetReady(ctx, awsNodeDS)

	// 	f.ResourceManager.WaitServiceHasEndpointsNum(ctx, awsNodeS, int(awsNodeDS.Status.NumberReady))
	// 	nodes, err = getTestNodes()
	// 	Expect(err).ShouldNot(HaveOccurred())
	// 	createNginxResources(nodes)

	// 	By("scaling testpod up to 25 pods")
	// 	nginxResourcesGroup.ExpectDeploymentScaleSuccessful(ctx, f, ns, 25)
	// 	time.Sleep(time.Second * 6)
	// 	testENIInfo(nodes, 2, ipLimit-1)

	// 	By("scaling testpod up to 26 pods")
	// 	nginxResourcesGroup.ExpectDeploymentScaleSuccessful(ctx, f, ns, 26)
	// 	time.Sleep(time.Second * 6) // TODO handle this because the ENI info is slower to load
	// 	testENIInfo(nodes, 3, ipLimit-1)
	// })

	AfterEach(func() {
		// promResources.ExpectCleanupSuccessful(ctx, f, ns)
		for _, resources := range resourcesGroup {
			// TODO log pods if they're not successful and maybe don't delete
			resources.ExpectCleanupSuccessful(ctx, f, ns)
		}
		// awsNodeSvc.ExpectCleanupSuccessful(ctx, f, kubeSystem)
	})
})

var _ = SynchronizedAfterSuite(func() {
	// Run on all Ginkgo nodes
	utils.Logf("Running AfterSuite actions on all nodes")
	framework.RunCleanupActions()
}, func() {
	Expect(f).NotTo(BeNil())
	// nginxResourcesGroup.ExpectCleanupSuccessful(ctx, f, ns)

	// testpodResources.ExpectDeploySuccessful(ctx, f, ns)
})
