package testpod

import (
	"crypto/tls"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"net"
	"net/http"
	"time"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

var (
	received = promauto.NewCounter(prometheus.CounterOpts{
		Name: "cni_test_received_total",
		Help: "Number of events received",
	})

	dnsRequests = promauto.NewCounter(prometheus.CounterOpts{
		Name: "cni_test_dns_request_total",
		Help: "Number of dns requests sent",
	})
	dnsRequestFailures = promauto.NewCounter(prometheus.CounterOpts{
		Name: "cni_test_dns_request_failure",
		Help: "Number of dns request failures",
	})

	externalHTTPRequests = promauto.NewCounter(prometheus.CounterOpts{
		Name: "cni_test_external_http_request_total",
		Help: "Number of external http requests sent",
	})
	externalHTTPRequestFailures = promauto.NewCounter(prometheus.CounterOpts{
		Name: "cni_test_external_http_request_failure",
		Help: "Number of external http request failures",
	})

	svcClusterIPRequests = promauto.NewCounter(prometheus.CounterOpts{
		Name: "cni_test_cluster_ip_request_total",
		Help: "Number of requests set to service's cluster IP",
	})
	svcClusterIPRequestFailures = promauto.NewCounter(prometheus.CounterOpts{
		Name: "cni_test_cluster_ip_request_failure",
		Help: "Number of requests that failed to reach cluster IP",
	})

	svcPodIPRequests = promauto.NewCounter(prometheus.CounterOpts{
		Name: "cni_test_pod_ip_request_total",
		Help: "Number of requests set to service's pod IP",
	})
	svcPodIPRequestFailures = promauto.NewCounter(prometheus.CounterOpts{
		Name: "cni_test_pod_ip_request_failure",
		Help: "Number of requests that failed to reach pod IP",
	})

	requests = promauto.NewCounterVec(prometheus.CounterOpts{
		Name: "cni_test_request_total",
		Help: "Number of total requests",
	}, []string{"pod_name", "pod_ip", "host_ip"})
	requestFailures = promauto.NewCounterVec(prometheus.CounterOpts{
		Name: "cni_test_request_failure",
		Help: "Number of successful requests",
	}, []string{"pod_name", "pod_ip", "host_ip"})
)

var (
	namespace = "cni-test"
)

func runServer() {
	http.HandleFunc("/healthz", func(w http.ResponseWriter, req *http.Request) {
		io.WriteString(w, "ok\n")
	})
	http.Handle("/metrics", promhttp.Handler())
	http.HandleFunc("/test", func(w http.ResponseWriter, req *http.Request) {
		received.Inc()
		io.WriteString(w, "ok\n")
	})
	s := http.Server{
		Addr:         ":8080",
		ReadTimeout:  10 * time.Second,
		WriteTimeout: 10 * time.Second,
	}
	log.Fatal(s.ListenAndServe())
}

func runTest() {
	// replace with operator-framework?
	config, err := rest.InClusterConfig()
	if err != nil {
		panic(err.Error())
	}
	// creates the clientset
	clientset, err := kubernetes.NewForConfig(config)
	if err != nil {
		panic(err.Error())
	}

	for {
		pods, err := clientset.CoreV1().Pods(namespace).List(metav1.ListOptions{
			LabelSelector: "app=testpod",
		})

		if err != nil {
			// TODO: this is terrible
			log.Print(err)
		}

		fmt.Printf("There are %d pods in the cluster\n", len(pods.Items))
		// TODO: wait for pods to be running -- this skip isn't working
		for _, pod := range pods.Items {
			if pod.Status.Phase != "Running" {
				log.Printf("Skipping pod %s in phase %s\n", pod.Name, pod.Status.Phase)
			}
			log.Printf("%+v\n", pod)
			counter := requests.WithLabelValues(pod.Name, pod.Status.PodIP, pod.Status.HostIP)
			failure := requestFailures.WithLabelValues(pod.Name, pod.Status.PodIP, pod.Status.HostIP)
			resp, err := http.Get(fmt.Sprintf("http://%s:8080/test", pod.Status.PodIP))
			counter.Inc()
			if err != nil {
				log.Printf("http error: %+v\n", err)
				failure.Inc()
				continue
			}
			data, err := ioutil.ReadAll(resp.Body)
			resp.Body.Close()
			if err != nil {
				log.Printf("Read error: %+v\n", err)
				failure.Inc()
				continue
			}
			s := string(data)
			if s != "ok\n" {
				log.Printf("Data was not 'ok': %s\n", s)
				failure.Inc()
				continue
			}
		}

		dnsRequests.Inc()
		_, err = net.LookupIP("google.com")
		if err != nil {
			log.Printf("dns lookup error: %+v\n", err)
			dnsRequestFailures.Inc()
		}

		externalHTTPRequests.Inc()
		httpClient := &http.Client{
			Transport: &http.Transport{
				TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
			},
		}
		_, err = httpClient.Get("https://www.google.com")
		if err != nil {
			log.Printf("http request to google.com failed: %+v\n", err)
			externalHTTPRequestFailures.Inc()
		}

		svcClusterIPRequests.Inc()
		resp, err := http.Get("http://testpod-clusterip.cni-test.svc.cluster.local:8080/healthz")
		if err != nil {
			log.Printf("http request to testpod-clusterip failed: %+v\n", err)
			svcClusterIPRequestFailures.Inc()
		} else {
			resp.Body.Close()
		}

		svcPodIPRequests.Inc()
		resp, err = http.Get("http://testpod-pod-ip.cni-test.svc.cluster.local:8080/healthz")
		if err != nil {
			log.Printf("http request to testpod-pod-ip failed: %+v\n", err)
			svcPodIPRequestFailures.Inc()
		} else {
			resp.Body.Close()
		}

		time.Sleep(time.Second)
	}

}

func main() {
	c := make(chan bool)
	go runServer()
	go runTest()
	<-c
}
