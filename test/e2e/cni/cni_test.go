package cni_test

import (
	"context"
	"time"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	promv1 "github.com/prometheus/client_golang/api/prometheus/v1"
	// _ "github.com/aws/amazon-vpc-cni-k8s/test/e2e/awsnode"
	"github.com/aws/amazon-vpc-cni-k8s/test/e2e/framework"
	"github.com/aws/amazon-vpc-cni-k8s/test/e2e/cni/shared"

	// "k8s.io/kubernetes/test/e2e/framework"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// Timeout for waiting events in seconds
// const TIMEOUT = 60
var (
	f  *framework.Framework
	ns *corev1.Namespace
)

// var zero model.SampleValue

// prom holds the created prom v1 API and the time the test runs
type prom struct {
	api      promv1.API
	testTime time.Time
}

// func TestCNI(t *testing.T) {
// 	RegisterFailHandler(Fail) //Make sure this works
// 	RunSpecs(t, "cni-tester") // TODO: see what this does
//

var _ = Describe("cni-tester", func() {
	f := framework.New()
	promReplicas := int32(1)
	ns := &corev1.Namespace{ObjectMeta: metav1.ObjectMeta{Name: "cni-test"}}

	var (
		ctx  context.Context
		prom *Resources
	)

	BeforeEach(func() {
		ctx = context.Background()
		prom = shared.NewPromResources(promReplicas)
		prom.ExpectDeploymentSuccessful(ctx, f, ns)
	})

	It("Should get 2 ENIs", func() {
		attachedENIs, err := f.AWSClient.GetAttachedENIs()
		Expect(err).ShouldNot(HaveOccurred())
		maxENIs, err := f.AWSClient.GetENILimit()
		Expect(err).ShouldNot(HaveOccurred())
		Expect(len(attachedENIs)).To(BeNumerically("<", maxENIs))
		Expect(len(attachedENIs)).To(Equal(2))

	})

	AfterEach(func() {
		prom.ExpectCleanupSuccessful(ctx, f, ns)
	})
})

// prom.ExpectCleanupSuccessfully(ctx, f, ns)
// var (
// 	clientset kubernetes.Interface
// 	pod       *corev1.Pod
// 	count     uint64 = 0
// 	replicas  int32  = 1
// 	mode      int32  = 420
// 	// var ns = "default"
// 	dep *appsv1.Deployment
// )

// BeforeEach(func() {
// 	// kubeconfig := os.Getenv("KUBECONFIG")
// 	// config, err := clientcmd.BuildConfigFromFlags("", kubeconfig)
// 	// Expect(err).ShouldNot(HaveOccurred())

// 	// // Create kubernetes client
// 	// client, err = kubernetes.NewForConfig(config)
// 	// Expect(err).ShouldNot(HaveOccurred())
// 	var kubeconfig *string
// 	if home := homedir.HomeDir(); home != "" {
// 		kubeconfig = flag.String("kubeconfig", filepath.Join(home, ".kube", "config"), "(optional) absolute path to the kubeconfig file")
// 	} else {
// 		kubeconfig = flag.String("kubeconfig", "", "absolute path to the kubeconfig file")
// 	}
// 	flag.Parse()

// 	config, err := clientcmd.BuildConfigFromFlags("", *kubeconfig)
// 	if err != nil {
// 		panic(err)
// 	}
// 	clientset, err := kubernetes.NewForConfig(config)
// 	if err != nil {
// 		panic(err)
// 	}

// 	newPromResources()

// })

// Context("blah", func() {
// 	testTime := time.Now()
// 	limit := 0.05 // TODO print this out?

// 	address := "http://localhost:9090"
// 	_, err := http.Get(address)
// 	It("should be nil", func() { //TODO edit caption
// 		Expect(err).To(BeNil()) // check this vs notto have occurred
// 	})

// 	cfg := promapi.Config{Address: address}
// 	client, err := promapi.NewClient(cfg)
// 	It("should be nil", func() { //TODO edit caption
// 		Expect(err).NotTo(HaveOccurred())
// 	})

// 	promAPI := promv1.NewAPI(client) // TODO does it exit from here if this fails?
// 	prom := &prom{
// 		api:      promAPI,
// 		testTime: testTime,
// 	}

// 	// // TODO: div by zero check?
// 	It("should get number of events received", func() {
// 		// TODO: set it for some # of expected requests?
// 		received, err := promAPI.Query(context.Background(), "cni_test_received_total", testTime)
// 		Expect(err).NotTo(HaveOccurred())
// 		Expect(received).NotTo(BeNil())
// 	})
// })
