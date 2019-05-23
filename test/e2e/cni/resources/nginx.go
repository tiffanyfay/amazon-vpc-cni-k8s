package resources

import (
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func NewNginxResources(replicas int32) *Resources {
	labels := map[string]string{
		"app": "nginx",
	}

	dp := &appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "nginx",
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
					ServiceAccountName: "testpod", // TODO change this name
					Containers: []corev1.Container{
						{
							Name:  "nginx",
							Image: "nginx:1.7.9",
							Ports: []corev1.ContainerPort{
								{
									ContainerPort: 80,
								},
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
			Name: "nginx",
		},
		Spec: corev1.ServiceSpec{
			// Type: svcType,
			Selector: map[string]string{
				"app": "nginx",
			},
			Ports: []corev1.ServicePort{
				{
					Port: 80,
				},
			},
		},
	}

	return &Resources{
		Deployment: dp,
		Service:    svc,
	}
}
