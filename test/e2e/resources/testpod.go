package resources

import (
	"os"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func NewTestpodResources(ns string, replicas int32) *Resources {
	app := "testpod"
	labels := map[string]string{
		"app": "testpod",
	}

	annotations := map[string]string{
		"prometheus.io/scrape": "true",
		"prometheus.io/port":   "8080",
	}

	dp := &appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Name:      app,
			Namespace: ns,
		},
		Spec: appsv1.DeploymentSpec{
			Replicas: &replicas,
			Selector: &metav1.LabelSelector{MatchLabels: labels}, //TODO check this
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Labels:      labels,
					Annotations: annotations,
				},
				Spec: corev1.PodSpec{
					ServiceAccountName: "testpod", // TODO change this name
					Containers: []corev1.Container{
						{
							Name:  app,
							Image: os.Getenv("TESTPOD_IMAGE_URI"),
							Ports: []corev1.ContainerPort{
								{
									Name:          "http",
									ContainerPort: 8080,
								},
							},
							ImagePullPolicy: corev1.PullAlways,
						},
					},
				},
			},
			Strategy: appsv1.DeploymentStrategy{
				Type: appsv1.RollingUpdateDeploymentStrategyType,
				RollingUpdate: &appsv1.RollingUpdateDeployment{
					MaxUnavailable: 1,
					MaxSurge:       5,
				},
			},
		},
	}

	svcs := []*coreV1.Service{}
	// svcType := corev1.ServiceTypeNodePort
	svcClusterIP := &corev1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "testpod-clusterip",
			Namespace: ns,
		},
		Spec: corev1.ServiceSpec{
			// Type: svcType,
			Selector: map[string]string{
				"app": app,
			},
			Ports: []corev1.ServicePort{
				{
					Port: 8080,
				},
			},
		},
	}

	svcPodIP := &corev1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "testpod-pod-ip",
			Namespace: ns,
		},
		Spec: corev1.ServiceSpec{
			// Type: svcType,
			Selector: map[string]string{
				"app": app,
			},
			ClusterIP: "None",
			Ports: []corev1.ServicePort{
				{
					Port: 8080,
				},
			},
		},
	}

	return &Resources{
		Deployment: dp,
		Services:   svcs,
	}
}
