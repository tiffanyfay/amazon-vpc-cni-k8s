package resources

import (
	batchv1 "k8s.io/api/batch/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func NewE2EJob(ns, image string) *batchv1.Job {
	name := "cni-e2e"
	labels := map[string]string{
		"app": name,
	}

	job := &batchv1.Job{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: ns,
		},
		Spec: batchv1.JobSpec{
			Selector: &metav1.LabelSelector{MatchLabels: labels}, //TODO check this
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Labels: labels,
				},
				Spec: corev1.PodSpec{
					ServiceAccountName: "testpod", // TODO change this name
					Containers: []corev1.Container{
						{
							Name:    name,
							Image:   image,
							Command: []string{"ginkgo", "-v", "test/e2e/cni"},
							// Ports: []corev1.ContainerPort{
							// 	{
							// 		ContainerPort: 80,
							// 	},
							// },
							ImagePullPolicy: corev1.PullAlways,
						},
					},
					RestartPolicy: corev1.RestartPolicyOnFailure,
				},
			},
		},
	}

	return job

	// 	return &Resources{
	// 		Deployment: dp,
	// 		Service:    svc,
	// 	}
}
