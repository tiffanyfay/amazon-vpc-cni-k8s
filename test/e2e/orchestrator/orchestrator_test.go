package orchestrator_test

import (
	"context"
	"time"

	"github.com/aws/amazon-vpc-cni-k8s/test/e2e/framework"
	"github.com/aws/amazon-vpc-cni-k8s/test/e2e/resources"

	log "github.com/cihub/seelog"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	promv1 "github.com/prometheus/client_golang/api/prometheus/v1"
	batchv1 "k8s.io/api/batch/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// Timeout for waiting events in seconds
// const TIMEOUT = 60
var (
	f             *framework.Framework
	ns            *corev1.Namespace
	promResources *resources.Resources
	prom          *resources.Prom
	promAPI       promv1.API
	count         int
)

var _ = Describe("cni-tester-prom", func() {
	f := framework.New()
	promReplicas := int32(1)
	ns = &corev1.Namespace{ObjectMeta: metav1.ObjectMeta{Name: "cni-test"}}

	var (
		ctx context.Context
		// conn net.Conn
	)

	BeforeEach(func() {
		log.Infof("count %d", count)
		count++
		var err error
		ctx = context.Background()
		// ns, err = f.ResourceManager.CreateNamespace(context.TODO(), "cni-test")
		// Expect(err).NotTo(HaveOccurred())

		promResources = resources.NewPromResources(ns.Name, promReplicas)
		promResources.ExpectDeploymentSuccessful(ctx, f, ns)

		promAPI, err = resources.NewPromAPI(f, ns)
		Expect(err).NotTo(HaveOccurred()) // TODO make sure this kills the test

		time.Sleep(time.Second * 5)
		testTime := time.Now() // TODO delete

		prom = &resources.Prom{
			API:      promAPI,
			TestTime: testTime, //TODO change
		}
	})

	It("awsCNIAWSAPIErrorCount should be 0", func() {
		query, err := prom.Query("awscni_aws_api_error_count")
		Expect(err).NotTo(HaveOccurred())
		Expect(query).NotTo(BeNil())
		Expect(query).To(BeNumerically("=", 0))
	})

	AfterEach(func() {
		promResources.ExpectCleanupSuccessful(ctx, f, ns)
	})
})

var _ = Describe("cni-tester-nodes", func() {
	f := framework.New()
	jobs := map[string]*batchv1.Job{}
	image := "038954622175.dkr.ecr.us-west-2.amazonaws.com/amazon-k8s-cni-e2e:latest"

	It("Should launch test jobs", func() {
		// Get nodes
		nodeList, err := f.ClientSet.CoreV1().Nodes().List(metav1.ListOptions{})
		Expect(err).NotTo(HaveOccurred())

		numNodes := nodeList.Size()
		Expect(numNodes).To(Equal(BeNumerically(">", 0)))

		// Launch pods per node with ginkgo cni test
		for _, node := range nodeList.Items {
			jobs[node.ObjectMeta.Name] = resources.NewE2EJob(ns.Name, image)
		}
		Expect(len(jobs)).To(Equal(numNodes))
		// TODO actually launch jobs then wait for them to finish then somehow get results
	})
})
