package shared

import (
	"context"
	"fmt"
	"time"

	"github.com/prometheus/common/model"

	// _ "github.com/aws/amazon-vpc-cni-k8s/test/e2e/awsnode"
	promv1 "github.com/prometheus/client_golang/api/prometheus/v1"
	// "k8s.io/kubernetes/test/e2e/framework"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// prom holds the created prom v1 API and the time the test runs
type Prom struct {
	API      promv1.API
	TestTime time.Time
}

func NewPromResources(replicas int32) *Resources {
	mode := int32(420)

	labels := map[string]string{
		"app": "prometheus-server",
	}

	dp := &appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "prometheus-deployment",
			Namespace: "cni-test",
		},
		Spec: appsv1.DeploymentSpec{
			Replicas: &replicas,
			Selector: &metav1.LabelSelector{MatchLabels: labels}, //TODO check this
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Labels: labels,
				},
				Spec: corev1.PodSpec{
					ServiceAccountName: "testpod",
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

	return &Resources{
		Deployment: dp,
		Service:    svc,
	}
}

func (p *Prom) Query(requests string, failures string) (model.SampleValue, error) {
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
	query, err := p.API.Query(context.Background(),
		fmt.Sprintf("sum(%s) / sum(%s)", failures, requests), p.TestTime)
	if err != nil {
		return 0, err
	}
	if len(query.(model.Vector)) != 1 {
		return 0, fmt.Errorf("query sum(%s) has no data at time %v", percent, p.TestTime)
	}
	return query.(model.Vector)[0].Value, err
}
