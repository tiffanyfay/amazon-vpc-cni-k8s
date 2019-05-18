package shared

import (
	"context"
	"time"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	promv1 "github.com/prometheus/client_golang/api/prometheus/v1"
	// _ "github.com/aws/amazon-vpc-cni-k8s/test/e2e/awsnode"
	"github.com/aws/amazon-vpc-cni-k8s/test/e2e/framework"

	// "k8s.io/kubernetes/test/e2e/framework"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// prom holds the created prom v1 API and the time the test runs
type prom struct {
	api      promv1.API
	testTime time.Time
}

// Add before and after for setup and delete of pods
type Resources struct {
	Deployment *appsv1.Deployment
	Service    *corev1.Service
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
			Selector: &metav1.LabelSelector{MatchLabels: labels},
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Labels: map[string]string{
						"app": "prometheus-server",
					},
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

func (r *Resources) ExpectDeploymentSuccessful(ctx context.Context, f *framework.Framework, ns *corev1.Namespace) {
	By("create deployment")
	dp, err := f.ClientSet.AppsV1().Deployments(ns.Name).Create(r.Deployment)
	Expect(err).NotTo(HaveOccurred())

	By("create service")
	svc, err := f.ClientSet.CoreV1().Services(ns.Name).Create(r.Service)
	Expect(err).NotTo(HaveOccurred())

	By("wait deployment")
	dp, err = f.ResourceManager.WaitDeploymentReady(ctx, dp)
	Expect(err).NotTo(HaveOccurred())

	By("wait service")
	_, err = f.ResourceManager.WaitServiceHasEndpointsNum(ctx, svc, int(*dp.Spec.Replicas))
	Expect(err).NotTo(HaveOccurred())
}

func (r *Resources) ExpectCleanupSuccessful(ctx context.Context, f *framework.Framework, ns *corev1.Namespace) {
	By("delete service")
	err := f.ClientSet.CoreV1().Services(ns.Name).Delete(r.Service.Name, &metav1.DeleteOptions{})
	Expect(err).NotTo(HaveOccurred())

	By("delete deployment")
	err = f.ClientSet.AppsV1().Deployments(ns.Name).Delete(r.Deployment.Name, &metav1.DeleteOptions{})
	Expect(err).NotTo(HaveOccurred())
}
