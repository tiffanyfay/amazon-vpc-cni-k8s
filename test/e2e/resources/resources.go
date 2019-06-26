package resources

import (
	"context"

	"github.com/aws/amazon-vpc-cni-k8s/test/e2e/framework"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// Add before and after for setup and delete of pods
// TODO

type Resources struct {
	Deployment *appsv1.Deployment
	Services   []*corev1.Service
}

// TODO method comment
func (r *Resources) ExpectDeploymentSuccessful(ctx context.Context, f *framework.Framework, ns *corev1.Namespace) {
	By("create deployment")
	dp, err := f.ClientSet.AppsV1().Deployments(ns.Name).Create(r.Deployment)
	Expect(err).NotTo(HaveOccurred())

	By("wait deployment")
	dp, err = f.ResourceManager.WaitDeploymentReady(ctx, dp)
	Expect(err).NotTo(HaveOccurred())

	By("create service(s)")
	for _, service := range r.Services {
		svc, err := f.ClientSet.CoreV1().Services(ns.Name).Create(service)
		Expect(err).NotTo(HaveOccurred())
		By("wait service")
		_, err = f.ResourceManager.WaitServiceHasEndpointsNum(ctx, svc, int(*dp.Spec.Replicas))
		Expect(err).NotTo(HaveOccurred())
	}
}

// TODO method comment
func (r *Resources) ExpectCleanupSuccessful(ctx context.Context, f *framework.Framework, ns *corev1.Namespace) {
	By("delete service(s)")
	for _, svc := range r.Services {
		err := f.ClientSet.CoreV1().Services(ns.Name).Delete(svc.Name, &metav1.DeleteOptions{})
		Expect(err).NotTo(HaveOccurred())
	}

	By("delete deployment")
	err := f.ClientSet.AppsV1().Deployments(ns.Name).Delete(r.Deployment.Name, &metav1.DeleteOptions{})
	Expect(err).NotTo(HaveOccurred())
}

// // TODO method comment
// func (r *Resources) ExpectDeploymentScaleSuccessful(ctx context.Context, f *framework.Framework, ns *corev1.Namespace, replicas int32) {
// 	By("scale deployment")
// 	scale, err := f.ClientSet.AppsV1().Deployments(ns.Name).GetScale(r.Deployment.Name, &metav1.GetOptions{})
// 	scale.Spec.Replicas = replicas
// 	scale, err = f.ClientSet.AppsV1().Deployments(ns.Name).UpdateScale(r.Deployment.Name, &scale)

// 	By("wait deployment")
// 	dp, err = f.ResourceManager.WaitDeploymentReady(ctx, r.Deployment)
// 	Expect(err).NotTo(HaveOccurred())
// }
