package resources

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"time"

	"github.com/aws/amazon-vpc-cni-k8s/test/e2e/framework"
	log "github.com/cihub/seelog"
	promapi "github.com/prometheus/client_golang/api"
	promv1 "github.com/prometheus/client_golang/api/prometheus/v1"
	"github.com/prometheus/common/model"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const (
	PromDeploymentName = "prometheus"
	PromServiceName    = "prometheus"
)

// prom holds the created prom v1 API and the time the test runs
type Prom struct {
	API      promv1.API
	TestTime time.Time
}

func NewPromResources(ns string, replicas int32) *Resources {
	mode := int32(420)

	labels := map[string]string{
		"app": "prometheus-server",
	}

	dp := &appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Name:      PromDeploymentName,
			Namespace: ns,
		},
		Spec: appsv1.DeploymentSpec{
			Replicas: &replicas,
			Selector: &metav1.LabelSelector{MatchLabels: labels}, //TODO check this
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Labels: labels,
				},
				Spec: corev1.PodSpec{
					ServiceAccountName: "testpod", // TODO change this name
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
									LocalObjectReference: corev1.LocalObjectReference{
										Name: "prometheus-server-conf",
									},
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

	svcs := []*corev1.Service{}
	// svcType := corev1.ServiceTypeNodePort
	svc := &corev1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name:      PromServiceName,
			Namespace: ns,
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

	svcs = append(svcs, svc)

	return &Resources{
		Deployment: dp,
		Services:   svcs,
	}
}

func NewPromAPI(f *framework.Framework, ns *corev1.Namespace) (promv1.API, error) {
	var resp *http.Response

	podList, err := f.ClientSet.CoreV1().Pods(ns.Name).List(metav1.ListOptions{
		LabelSelector: "app=prometheus-server",
	})
	if err != nil {
		return nil, err
	}

	if len(podList.Items) == 0 {
		return nil, errors.New("Error getting prometheus pod(s)")
	}

	// Check if prometheus is healthy
	address := fmt.Sprintf("http://%s.%s.svc.cluster.local:9090", PromServiceName, ns.Name)
	health := fmt.Sprintf("%s/-/healthy", address)

	for i; i < 3; i++ {
		resp, err = http.Get(health)
		if err == nil {
			break
		}
		time.Sleep(time.Second * 5)
	}
	if err != nil {
		return nil, err
	}
	resp.Body.Close()
	log.Infof("healthy %v %v", resp.StatusCode, resp.Status)
	// TODO maybe handle .Status
	if resp.StatusCode != 200 {
		return nil, errors.New("prometheus is not healthy")
	}

	// Create prometheus client and API
	cfg := promapi.Config{Address: address}
	client, err := promapi.NewClient(cfg)
	if err != nil {
		return nil, err
	}
	return promv1.NewAPI(client), nil
}

// TODO
func (p *Prom) QueryPercent(requests string, failures string) (model.SampleValue, error) {
	// if either is 0 return 0
	requestsQuery, err := p.API.Query(context.Background(),
		fmt.Sprintf("sum(%s)", requests), p.TestTime)
	if err != nil {
		return 0, fmt.Errorf("query sum(%s) has value of 0 at time %v", requests, p.TestTime)
	}
	if len(requestsQuery.(model.Vector)) != 1 {
		return 0, fmt.Errorf("query sum(%s) has no data at time %v", requests, p.TestTime)
	}

	failuresQuery, err := p.API.Query(context.Background(),
		fmt.Sprintf("sum(%s)", failures), p.TestTime)
	if err != nil {
		return 0, err
	}
	if len(failuresQuery.(model.Vector)) != 1 {
		return 0, fmt.Errorf("query sum(%s) has no data at time %v", failures, p.TestTime)
	}
	if failuresQuery.(model.Vector)[0].Value == 0 { //todo make sure the check works
		return 0, nil
	}

	percent := fmt.Sprintf("sum(%s) / sum(%s)", failures, requests)
	percentQuery, err := p.API.Query(context.Background(),
		fmt.Sprintf("sum(%s) / sum(%s)", failures, requests), p.TestTime)
	if err != nil {
		return 0, err
	}
	if len(percentQuery.(model.Vector)) != 1 {
		return 0, fmt.Errorf("query sum(%s) has no data at time %v", percent, p.TestTime)
	}
	return percentQuery.(model.Vector)[0].Value, err
}

// TODO
func (p *Prom) Query(name string) (model.SampleValue, error) {
	// if either is 0 return 0
	query, err := p.API.Query(context.Background(), name, p.TestTime)
	if err != nil {
		return 0, err
	}
	if len(query.(model.Vector)) != 1 {
		return 0, fmt.Errorf("query (%s) has no data at time %v", p.TestTime)
	}
	return query.(model.Vector)[0].Value, err
}
