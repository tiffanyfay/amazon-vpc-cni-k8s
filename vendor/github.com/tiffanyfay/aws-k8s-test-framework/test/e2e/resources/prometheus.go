package resources

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"time"

	log "github.com/cihub/seelog"
	promapi "github.com/prometheus/client_golang/api"
	promv1 "github.com/prometheus/client_golang/api/prometheus/v1"
	"github.com/prometheus/common/model"
	"github.com/tiffanyfay/aws-k8s-test-framework/test/e2e/framework"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const (
	PromDeploymentName = "prometheus"
	PromServiceName    = "prometheus"
	PromImage          = "prom/prometheus:v2.1.0"
)

// Prom holds the created prom v1 API and the time the test runs
type Prom struct {
	API promv1.API
}

// NewPromResources creates new prometheus Kubernetes resources and takes in a namespace,
// the node name to run on, and replica count
func NewPromResources(ns string, nodeName string, replicas int32) *Resources {
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
					ServiceAccountName: "cni-tester", // TODO change this name
					Affinity: &corev1.Affinity{
						NodeAffinity: &corev1.NodeAffinity{
							RequiredDuringSchedulingIgnoredDuringExecution: &corev1.NodeSelector{
								NodeSelectorTerms: []corev1.NodeSelectorTerm{
									{
										MatchExpressions: []corev1.NodeSelectorRequirement{
											{
												Key:      "kubernetes.io/hostname",
												Operator: corev1.NodeSelectorOpIn,
												Values:   []string{nodeName},
											},
										},
									},
								},
							},
						},
					},
					Containers: []corev1.Container{
						{
							Name:  "prometheus",
							Image: PromImage,
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

	svc := &corev1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name:      PromServiceName,
			Namespace: ns,
		},
		Spec: corev1.ServiceSpec{
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

// NewPromAPI creates a new Prometheus API
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

	for i := 0; i < 3; i++ {
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

// QueryPercent returns the percentage value for a Prometheus query
func (p *Prom) QueryPercent(requests string, failures string, testTime time.Time) (model.SampleValue, error) {
	// if either is 0 return 0
	requestsOut, warnings, err := p.API.Query(context.Background(),
		fmt.Sprintf("sum(%s)", requests), testTime)
	if err != nil {
		return 0, err
	}
	if len(warnings) > 0 {
		log.Debug(warnings)
	}
	if len(requestsOut.(model.Vector)) != 1 {
		log.Debugf("prometheus query sum(%s) has no data at time %v", requests, testTime)
		return 0, nil
	}
	if requestsOut.(model.Vector)[0].Value == 0 {
		return 0, nil
	}

	failuresOut, warnings, err := p.API.Query(context.Background(),
		fmt.Sprintf("sum(%s)", failures), testTime)
	if err != nil {
		return 0, err
	}
	if len(warnings) > 0 {
		log.Debug(warnings)
	}
	if len(failuresOut.(model.Vector)) != 1 {
		log.Debugf("prometheus query sum(%s) has no data at time %v", failures, testTime)
		return 0, nil
	}
	if failuresOut.(model.Vector)[0].Value == 0 {
		return 0, nil
	}

	query := fmt.Sprintf("sum(%s) / sum(%s)", failures, requests)
	out, warnings, err := p.API.Query(context.Background(),
		fmt.Sprintf("sum(%s) / sum(%s)", failures, requests), testTime)
	if err != nil {
		return 0, err
	}
	if len(warnings) > 0 {
		log.Debug(warnings)
	}
	if len(out.(model.Vector)) != 1 {
		return 0, fmt.Errorf("query (%s) has no data at time %v", query, testTime)
	}
	return out.(model.Vector)[0].Value, err
}

// Query returns the value for the Prometheus query
func (p *Prom) Query(query string, testTime time.Time) (model.SampleValue, error) {
	out, warnings, err := p.API.Query(context.Background(), query, testTime)
	if err != nil {
		return 0, err
	}
	if len(warnings) > 0 {
		log.Debug(warnings)
	}
	if len(out.(model.Vector)) != 1 {
		log.Debugf("prometheus query (%s) has no data at time %v", query, testTime)
		return 0, nil
	}
	return out.(model.Vector)[0].Value, err
}
