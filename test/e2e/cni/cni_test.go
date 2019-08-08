package cni_test

import (
	"context"
	"fmt"
	"time"

	"github.com/aws/amazon-vpc-cni-k8s/test/e2e/cni"
	"github.com/tiffanyfay/aws-k8s-test-framework/test/e2e/framework"
	"github.com/tiffanyfay/aws-k8s-test-framework/test/e2e/resources"

	log "github.com/cihub/seelog"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	promv1 "github.com/prometheus/client_golang/api/prometheus/v1"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// Timeout for waiting events in seconds// const TIMEOUT = 60
var (
	err                    error
	testerNodeName         string
	testPodErrPercentLimit float64
	awsNodeErrLimit        int64

	f                *framework.Framework
	prom             *resources.Prom
	promResources    *resources.Resources
	testpodResources *resources.Resources
	resourcesGroup   []*resources.Resources

	nodes   []corev1.Node
	promAPI promv1.API
)

var _ = Describe("Testing CNI", func() {
	f = framework.New()
	ctx := context.Background() // TODO make this have a timeout

	ns := &corev1.Namespace{ObjectMeta: metav1.ObjectMeta{Name: "cni-test"}}
	kubeSystem := &corev1.Namespace{ObjectMeta: metav1.ObjectMeta{Name: "kube-system"}}
	awsNodeDS := &appsv1.DaemonSet{ObjectMeta: metav1.ObjectMeta{Name: "aws-node", Namespace: kubeSystem.Name}}

	// testPodErrPercentLimit := 0.1
	awsNodeErrLimit = 5

	Context("Testing with 1 node per test", func() {
		It("Should wait for desired number of nodes", func() {
			desired := 3

			nodeList, err := f.ClientSet.CoreV1().Nodes().List(metav1.ListOptions{})
			Expect(err).ShouldNot(HaveOccurred())

			err = cni.WaitForASGDesiredCapacity(ctx, f, nodeList.Items, desired)
			Expect(err).ShouldNot(HaveOccurred())

			testerNodeName, err = cni.GetTesterPodNodeName(f, ns.Name, "cni-e2e")
			Expect(err).ShouldNot(HaveOccurred())

			nodes, err = cni.GetTestNodes(f, testerNodeName)
			Expect(err).ShouldNot(HaveOccurred())
			Expect(len(nodes)).To(BeNumerically(">=", desired-1))
		})
		Context("Run tests", func() {
			BeforeEach(func() {
				By("Creating prometheus resources")
				promResources = resources.NewPromResources(ns.Name, testerNodeName, 1)
				promResources.ExpectDeploySuccessful(ctx, f, ns)

				promAPI, err = resources.NewPromAPI(f, ns)
				Expect(err).NotTo(HaveOccurred())
				time.Sleep(time.Second * 5)

				prom = &resources.Prom{API: promAPI}
				time.Sleep(time.Second * 5)

				resourcesGroup = createTestResources(ctx, ns, nodes)
			})

			It("Should pass with WARM_IP_TARGET=0 (default), WARM_ENI_TARGET=1 (default), and MAX_ENI=-1(default)", func() {
				cni.UpdateAWSNodeEnvs(ctx, f, "0", "1", "-1")
				i := 0
				internalIP, _, ipLimit := setup(ctx, f, awsNodeDS, i)

				By(fmt.Sprintf("scaling deployment (%s) to get %d pods and 3 ENIs", resourcesGroup[i].Deployment.Name, ipLimit*2))
				resourcesGroup[i].ExpectDeploymentScaleSuccessful(ctx, f, ns, int32(ipLimit*2))
				cni.TestENIInfo(ctx, f, internalIP, 3, ipLimit)

				By(fmt.Sprintf("scaling deployment (%s) to get %d pods and 4 ENIs", resourcesGroup[i].Deployment.Name, ipLimit*2+1))
				resourcesGroup[i].ExpectDeploymentScaleSuccessful(ctx, f, ns, int32(ipLimit*2+1))
				cni.TestENIInfo(ctx, f, internalIP, 4, ipLimit)
			})

			It("Should pass with WARM_IP_TARGET=0 (default), WARM_ENI_TARGET=2, and MAX_ENI=-1(default)", func() {
				cni.UpdateAWSNodeEnvs(ctx, f, "0", "2", "-1")
				i := 1
				internalIP, _, ipLimit := setup(ctx, f, awsNodeDS, i)

				By(fmt.Sprintf("scaling deployment (%s) to get %d pods and 3 ENIs", resourcesGroup[i].Deployment.Name, ipLimit*2))
				resourcesGroup[i].ExpectDeploymentScaleSuccessful(ctx, f, ns, int32(ipLimit*2))
				cni.TestENIInfo(ctx, f, internalIP, 4, ipLimit)
			})

			AfterEach(func() {
				promResources.ExpectCleanupSuccessful(ctx, f, ns)
				for _, resources := range resourcesGroup {
					// TODO log pods if they're not successful and maybe don't delete
					resources.ExpectCleanupSuccessful(ctx, f, ns)
				}
			})
		})
	})
})

func createTestResources(ctx context.Context, ns *corev1.Namespace, nodes []corev1.Node) []*resources.Resources {
	Expect(len(nodes)).To(BeNumerically(">", 0))
	var resourcesGroup []*resources.Resources

	for i, node := range nodes {
		log.Infof("Creating deployment for node %d/%d: %v", i+1, len(nodes), node.Name)
		resource := resources.NewNginxResources(ns.Name, node.Name, 0)
		resource.ExpectDeploySuccessful(ctx, f, ns)
		resourcesGroup = append(resourcesGroup, resource)
	}
	return resourcesGroup
}

// setup gets instance limits, the node's internal IP address, and checks prometheus metrics
func setup(ctx context.Context, f *framework.Framework, ds *appsv1.DaemonSet, i int) (string, int, int) {
	f.ResourceManager.WaitDaemonSetReady(ctx, ds)

	testTime := time.Now()
	// TODO get testpod metrics per instance
	// testTestpodPromMetrics(testTime, testPodErrPercentLimit)

	eniLimit, ipLimit, err := cni.GetInstanceLimits(f, nodes[i].Name)
	Expect(err).ToNot(HaveOccurred())
	count, err := cni.NodeCoreDNSCount(f, nodes[i].Name)
	Expect(err).ToNot(HaveOccurred())
	ipLimit -= count
	internalIP, err := cni.GetNodeInternalIP(nodes[i])
	Expect(err).ToNot(HaveOccurred())
	promInstance := fmt.Sprintf("%s:61678", internalIP)

	testAWSNodePromMetrics(testTime, promInstance, eniLimit, ipLimit, awsNodeErrLimit)
	return internalIP, eniLimit, ipLimit
}

// TODO: maybe make it per instance
func testTestpodPromMetrics(testTime time.Time, errPercentLimit float64) {
	By("checking prometheus testpod number of events received", func() {
		// TODO: set it for some # of expected requests?
		receivedTotal, err := prom.Query("cni_test_received_total", testTime)
		log.Infof("receivedTotal %v", receivedTotal)
		Expect(err).NotTo(HaveOccurred())
		Expect(receivedTotal).NotTo(BeNil())
	})
	By("checking prometheus testpod dnsRequestFailurePercent", func() {
		dnsRequestFailurePercent, err := prom.QueryPercent("cni_test_dns_request_total",
			"cni_test_dns_request_failure", testTime)
		Expect(err).NotTo(HaveOccurred())
		Expect(dnsRequestFailurePercent).NotTo(BeNil())
		log.Infof("dnsRequestFailurePercent %v", dnsRequestFailurePercent)
		Expect(dnsRequestFailurePercent).To(BeNumerically("<", errPercentLimit))
	})
	By("checking prometheus testpod externalHTTPRequestsFailurePercent", func() {
		externalHTTPRequestsFailurePercent, err := prom.QueryPercent("cni_test_external_http_request_total",
			"cni_test_external_http_request_failure", testTime)
		Expect(err).NotTo(HaveOccurred())
		Expect(externalHTTPRequestsFailurePercent).NotTo(BeNil())
		Expect(externalHTTPRequestsFailurePercent).To(BeNumerically("<", errPercentLimit))
	})
	By("checking prometheus testpod svcClusterIPRequestFailurePercent", func() {
		svcClusterIPRequestFailurePercent, err := prom.QueryPercent("cni_test_cluster_ip_request_total",
			"cni_test_cluster_ip_request_failure", testTime)
		Expect(err).NotTo(HaveOccurred())
		Expect(svcClusterIPRequestFailurePercent).NotTo(BeNil())
		Expect(svcClusterIPRequestFailurePercent).To(BeNumerically("<", errPercentLimit))
	})
	By("checking prometheus testpod svcPodIPRequestsFailurePercent", func() {
		svcPodIPRequestsFailurePercent, err := prom.QueryPercent("cni_test_external_http_request_total",
			"cni_test_external_http_request_failure", testTime)
		Expect(err).NotTo(HaveOccurred())
		Expect(svcPodIPRequestsFailurePercent).NotTo(BeNil())
		Expect(svcPodIPRequestsFailurePercent).To(BeNumerically("<", errPercentLimit))
	})
}

// TODO: add more metrics
func testAWSNodePromMetrics(testTime time.Time, instanceName string, eniLimit int, ipLimit int, errLimit int64) {
	By(fmt.Sprintf("checking prometheus awscni_eni_max (%s)", instanceName), func() {
		out, err := prom.Query(fmt.Sprintf("awscni_eni_max{instance='%s'}", instanceName), testTime)
		Expect(err).NotTo(HaveOccurred())
		Expect(out).NotTo(BeNil())
		Expect(out).To(BeNumerically("==", eniLimit))
	})
	By(fmt.Sprintf("checking prometheus awscni_ip_max (%s)", instanceName), func() {
		out, err := prom.Query(fmt.Sprintf("awscni_ip_max{instance='%s'}", instanceName), testTime)
		Expect(err).NotTo(HaveOccurred())
		Expect(out).NotTo(BeNil())
		Expect(out).To(BeNumerically("==", eniLimit*ipLimit))
	})
	By(fmt.Sprintf("checking prometheus awscni_aws_api_error_count (%s)", instanceName), func() {
		out, err := prom.Query(fmt.Sprintf("awscni_aws_api_error_count{instance='%s'}", instanceName), testTime)
		Expect(err).NotTo(HaveOccurred())
		Expect(out).NotTo(BeNil())
		Expect(out).To(BeNumerically("<=", errLimit))
	})
}