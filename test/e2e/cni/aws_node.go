package cni

import (
	"context"
	"testing"
	"time"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	promv1 "github.com/prometheus/client_golang/api/prometheus/v1"

	"github.com/aws/amazon-vpc-cni-k8s/test/e2e/framework"
	_ "github.com/aws/amazon-vpc-cni-k8s/test/e2e/cni"
	// "k8s.io/kubernetes/test/e2e/framework"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// Timeout for waiting events in seconds
// const TIMEOUT = 60

// var zero model.SampleValue

// prom holds the created prom v1 API and the time the test runs
type prom struct {
	api      promv1.API
	testTime time.Time
}

func TestCNI(t *testing.T) {
	RegisterFailHandler(Fail) //Make sure this works
	RunSpecs(t, "cni-tester") // TODO: see what this does
}

// Add before and after for setup and delete of pods
type PromResources struct {
	Deployment *appsv1.Deployment
	Service    *corev1.Service
}

func newPromResources(replicas int32) *PromResources {
	mode := int32(420)

	dp := &appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "prometheus-deployment",
			Namespace: "cni-test",
		},
		Spec: appsv1.DeploymentSpec{
			Replicas: &replicas,
			// Selector: &metav1.LabelSelector{MatchLabels: stackLabels},
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Labels: map[string]string{
						"app": "prometheus-server",
					},
				},
				Spec: corev1.PodSpec{
					Containers: []corev1.Container{
						{
							Name:  "prometheus",
							Image: "prom/prometheus:v2.1.0",
							Ports: []corev1.ContainerPort{
								{
									ContainerPort: 9090,
								},
							},
							Args: []string{
								"--config.file=/etc/prometheus/prometheus.yml",
								"--storage.tsdb.path=/prometheus/",
							},
							VolumeMounts: []corev1.VolumeMount{
								{
									Name:      "prometheus-config-volume",
									MountPath: "/etc/prometheus/",
								},
								{
									Name:      "prometheus-storage-volume",
									MountPath: "/prometheus/",
								},
							},
						},
					},
					Volumes: []corev1.Volume{
						{
							Name: "prometheus-config-volume",
							VolumeSource: corev1.VolumeSource{
								ConfigMap: &corev1.ConfigMapVolumeSource{
									DefaultMode: &mode,
								},
							},
						},
						{
							Name: "prometheus-storage-volume",
							VolumeSource: corev1.VolumeSource{
								EmptyDir: &corev1.EmptyDirVolumeSource{},
							},
						},
					},
				},
			},
		},
	}

	// svcType := corev1.ServiceTypeNodePort
	svc := &corev1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name: "prometheus",
		},
		Spec: corev1.ServiceSpec{
			// Type: svcType,
			Selector: map[string]string{
				"app": "prometheus-server",
			},
			Ports: []corev1.ServicePort{
				{
					Port: 9090,
				},
			},
		},
	}

	return &PromResources{
		Deployment: dp,
		Service:    svc,
	}
}

func (r *PromResources) ExpectDeploymentSuccessful(ctx context.Context, f *framework.Framework, ns *corev1.Namespace) {
	By("create deployment")
	dp, err := f.ClientSet.AppsV1().Deployments(ns.Name).Create(r.Deployment)
	Expect(err).NotTo(HaveOccurred())

	By("create service")
	svc, err := f.ClientSet.CoreV1().Services(ns.Name).Create(r.Service)
	Expect(err).NotTo(HaveOccurred())

	By("wait deployment")
	dp, err = f.ResourceManager.WaitDeploymentReady(ctx, dp)
	Expect(err).NotTo(HaveOccurred())

	By("wait service")
	_, err = f.ResourceManager.WaitServiceHasEndpointsNum(ctx, svc, int(*dp.Spec.Replicas))
	Expect(err).NotTo(HaveOccurred())
}

var _ = Describe("cni-tester", func() {
	f := framewgo ork.New()

	var (
		ctx context.Context
		ns  *corev1.Namespace
	)

	BeforeEach(func() {
		ctx = context.Background()
		var err error
		ns, err = f.ResourceManager.CreateNamespaceUnique(context.TODO(), "cni-test")
		Expect(err).NotTo(HaveOccurred())
	})

	It("[mod-instance] should work", func() {
		prom := newPromResources(int32(1))
		// stackName := "multi-path-echo"
		// stack := NewMultiPathEchoStack(stackName, false)
		prom.ExpectDeploymentSuccessful(ctx, f, ns)
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
