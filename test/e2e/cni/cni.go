package cni_test

import (
	"context"
	"net"
	"net/http"
	"os"
	"time"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	promapi "github.com/prometheus/client_golang/api"
	promv1 "github.com/prometheus/client_golang/api/prometheus/v1"

	// _ "github.com/aws/amazon-vpc-cni-k8s/test/e2e/awsnode"
	"github.com/aws/amazon-vpc-cni-k8s/test/e2e/cni/resources"
	"github.com/aws/amazon-vpc-cni-k8s/test/e2e/framework"

	// "k8s.io/kubernetes/test/e2e/framework"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/client-go/tools/portforward"
	"k8s.io/client-go/transport/spdy"

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
)

var _ = Describe("cni-tester", func() {
	f := framework.New()
	promReplicas := int32(1)
	ns = &corev1.Namespace{ObjectMeta: metav1.ObjectMeta{Name: "cni-test"}}

	var (
		ctx context.Context
		// conn net.Conn
	)

	stopChan := make(chan struct{}, 1)
	readyChan := make(chan struct{})

	// It("portforward new should be nil", func() { //TODO edit caption
	// 	Expect(err).To(BeNil()) // check this vs notto have occurred
	// })

	BeforeEach(func() {
		var err error
		ctx = context.Background()
		// ns, err = f.ResourceManager.CreateNamespace(context.TODO(), "cni-test")
		// Expect(err).NotTo(HaveOccurred())

		promResources = resources.NewPromResources(promReplicas)
		promResources.ExpectDeploymentSuccessful(ctx, f, ns)

		podList, err := f.ClientSet.CoreV1().Pods(ns.Name).List(metav1.ListOptions{LabelSelector: "app=prometheus-server"})
		if err != nil {
			panic("Error listing prometheus pod(s)")
		}

		if len(podList.Items) == 0 {
			Fail("Error getting prometheus pod(s)")
		}
		podName := podList.Items[0].Name
		podNameSpace := podList.Items[0].Namespace

		// port forwarding
		go func() {
			req := f.ClientSet.CoreV1().RESTClient().Post().Resource("pods").
				Namespace(podNameSpace).Name(podName).SubResource("portforward")
			url := req.URL()
			transport, upgrader, err := spdy.RoundTripperFor(f.Config)
			if err != nil {
				Fail("Error getting roundtripper")
			}
			dialer := spdy.NewDialer(upgrader, &http.Client{Transport: transport}, "POST", url)
			ports := []string{"9090:9090"}

			fw, err := portforward.New(dialer, ports, stopChan, readyChan, nil, os.Stderr)
			if err != nil {
				Fail("Error creating new port-forwarding")
			}
			err = fw.ForwardPorts()
			if err != nil {
				Fail("Error port-forwarding")
			}
		}()
		for {
			conn, _ := net.DialTimeout("tcp",
				net.JoinHostPort("", "9090"), time.Millisecond)
			if conn != nil {
				conn.Close()
				break
			}
			time.Sleep(time.Millisecond * 50)
		}

		// TODO: change to svc
		address := "http://localhost:9090"
		_, err = http.Get(address)
		if err != nil {
			Fail("Unable to reach prometheus port")
		}

		cfg := promapi.Config{Address: address}
		client, err := promapi.NewClient(cfg)
		Expect(err).NotTo(HaveOccurred())

		promAPI = promv1.NewAPI(client) // TODO does it exit from here if this fails?
		time.Sleep(time.Second * 5)
		testTime := time.Now() // TODO delete
		prom = &resources.Prom{
			API:      promAPI,
			TestTime: testTime, //TODO change
		}
	})
	Context("prom should pass", func() {
		// It("should get no ipamd errors", func() {
		// 	log.Info("hi")

		// 	prom.TestTime = time.Now()
		// 	received, err := prom.Query(ctx, "awscni_ip_max", prom.TestTime)
		// 	Expect(err).NotTo(HaveOccurred())
		// 	// Expect(received).NotTo(BeNil())

		// 	log.Info(received.(model.Vector)[0].Value)
		// 	// Expect(received.String()).To(BeNumerically(">", 1))

		// 	dnsRequestFailurePercent, err := prom.Query("cni_test_external_http_request_total",
		// 		"cni_test_dns_request_failure")
		// 	Expect(err).NotTo(HaveOccurred())
		// 	Expect(dnsRequestFailurePercent).NotTo(BeNil())
		// 	Expect(dnsRequestFailurePercent).To(BeNumerically("<", 0.05))
		// })
		It("Should get 2 ENIs", func() {
			attachedENIs, err := f.AWSClient.GetAttachedENIs()
			Expect(err).ShouldNot(HaveOccurred())
			maxENIs, err := f.AWSClient.GetENILimit()
			Expect(err).ShouldNot(HaveOccurred())
			Expect(len(attachedENIs)).To(BeNumerically("<", maxENIs))
			Expect(len(attachedENIs)).To(Equal(2))

		})
	})

	// It("Should get 2 ENIs", func() {
	// 	attachedENIs, err := f.AWSClient.GetAttachedENIs()
	// 	Expect(err).ShouldNot(HaveOccurred())
	// 	maxENIs, err := f.AWSClient.GetENILimit()
	// 	Expect(err).ShouldNot(HaveOccurred())
	// 	Expect(len(attachedENIs)).To(BeNumerically("<", maxENIs))
	// 	Expect(len(attachedENIs)).To(Equal(2))

	// })

	AfterEach(func() {
		// conn.Close()
		close(stopChan)

		promResources.ExpectCleanupSuccessful(ctx, f, ns)
	})
})
