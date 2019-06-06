package cni_test

import (
	log "github.com/cihub/seelog"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	"github.com/aws/amazon-vpc-cni-k8s/test/e2e/framework"
	// corev1 "k8s.io/api/core/v1"
	// metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// Timeout for waiting events in seconds
// const TIMEOUT = 60
var (
	f *framework.Framework
	// ns *corev1.Namespace
	// promResources *resources.Resources
	// prom          *resources.Prom
	// promAPI       promv1.API
	// count int
)

var _ = Describe("cni-tester", func() {
	f := framework.New()
	// promReplicas := int32(1)
	// ns = &corev1.Namespace{ObjectMeta: metav1.ObjectMeta{Name: "cni-test"}}

	// var (
	// 	ctx context.Context
	// 	// conn net.Conn
	// )

	// 	// // TODO: change to svc
	// 	// address := "http://localhost:9090"
	// 	// _, err = http.Get(address)
	// 	// if err != nil {
	// 	// 	Fail("Unable to reach prometheus port")
	// 	// }

	// 	// svcList, err := f.ClientSet.CoreV1().Services(ns.Name).List(metav1.ListOptions{
	// 	// 	LabelSelector: "app=prometheus-server",
	// 	// })

	// 	address := fmt.Sprintf("http://%s.cni-test.svc.cluster.local:9090", resources.PromServiceName)
	// 	health := fmt.Sprintf("%s/-/healthy", address)

	// 	resp, err := http.Get(health)
	// 	if err != nil {
	// 		// Fail("http request to %s failed: %+v\n", svcName, err)
	// 		Fail("http request to prometheus failed")
	// 	} else {
	// 		resp.Body.Close()
	// 		log.Infof("healthy %v %v", resp.StatusCode, resp.Status)
	// 		if resp.StatusCode != 200 {
	// 			Fail("prometheus is not healthy")
	// 		}
	// 	}

	// 	cfg := promapi.Config{Address: address}
	// 	client, err := promapi.NewClient(cfg)
	// 	Expect(err).NotTo(HaveOccurred())

	// 	promAPI = promv1.NewAPI(client) // TODO does it exit from here if this fails?
	// 	time.Sleep(time.Second * 5)
	// 	testTime := time.Now() // TODO delete
	// 	prom = &resources.Prom{
	// 		API:      promAPI,
	// 		TestTime: testTime, //TODO change
	// 	}
	// 	log.Debug("prom resources")
	// })

	It("Should get 2 ENIs", func() {
		log.Debug("blah 2 enis")
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

	AfterEach(func() {
		// conn.Close()
		// close(stopChan)

		promResources.ExpectCleanupSuccessful(ctx, f, ns)
	})
})
