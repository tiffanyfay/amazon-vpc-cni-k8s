package resources

import (
	"fmt"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// NewNginxResources creates new Kubernetes nginx resources and takes in a namespace,
// the node name to run on, and replica count
func NewNginxResources(ns, nodeName string, replicas int32) *Resources {
	labels := map[string]string{
		"app": "nginx",
	}

	affinity := &corev1.Affinity{}
	if nodeName != "" {
		affinity = &corev1.Affinity{
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
		}
	}

	dp := &appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Name:      fmt.Sprintf("nginx-%s", nodeName),
			Namespace: ns,
		},
		Spec: appsv1.DeploymentSpec{
			Replicas: &replicas,
			Selector: &metav1.LabelSelector{MatchLabels: labels},
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Labels: labels,
				},
				Spec: corev1.PodSpec{
					ServiceAccountName: "cni-tester",
					Affinity:           affinity,
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

	svcs := []*corev1.Service{}
	// svcType := corev1.ServiceTypeNodePort
	// svc := &corev1.Service{
	// 	ObjectMeta: metav1.ObjectMeta{
	// 		Name:      "nginx",
	// 		Namespace: ns,
	// 	},
	// 	Spec: corev1.ServiceSpec{
	// 		// Type: svcType,
	// 		Selector: map[string]string{
	// 			"app": "nginx",
	// 		},
	// 		Ports: []corev1.ServicePort{
	// 			{
	// 				Port: 80,
	// 			},
	// 		},
	// 	},
	// }

	// svcs = append(svcs, svc)

	return &Resources{
		Deployment: dp,
		Services:   svcs,
	}
}
