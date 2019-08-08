package resources

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/tiffanyfay/aws-k8s-test-framework/test/e2e/framework"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
)

// Add before and after for setup and delete of pods
// TODO

type Resources struct {
	Daemonset  *appsv1.DaemonSet
	Deployment *appsv1.Deployment
	Services   []*corev1.Service
}

type ResourcesGroup struct {
	ResourcesGroup []*Resources
}

// ExpectDeploySuccessful expects a deployment and any services to be successful
func (r *Resources) ExpectDeploySuccessful(ctx context.Context, f *framework.Framework, ns *corev1.Namespace) {
	By(fmt.Sprintf("create deployment (%s)", r.Deployment.Name))
	dp, err := f.ClientSet.AppsV1().Deployments(ns.Name).Create(r.Deployment)
	Expect(err).NotTo(HaveOccurred())

	By(fmt.Sprintf("wait deployment (%s)", r.Deployment.Name))
	dp, err = f.ResourceManager.WaitDeploymentReady(ctx, dp)
	Expect(err).NotTo(HaveOccurred())

	for _, service := range r.Services {
		By(fmt.Sprintf("create service (%s)", service.Name))
		svc, err := f.ClientSet.CoreV1().Services(ns.Name).Create(service)
		Expect(err).NotTo(HaveOccurred())
		By(fmt.Sprintf("wait service (%s)", service.Name))
		_, err = f.ResourceManager.WaitServiceHasEndpointsNum(ctx, svc, int(*dp.Spec.Replicas))
		Expect(err).NotTo(HaveOccurred())
	}
}

// ExpectCleanupSuccessful expects cleaning up services and deployments to be successful
func (r *Resources) ExpectCleanupSuccessful(ctx context.Context, f *framework.Framework, ns *corev1.Namespace) {
	for _, service := range r.Services {
		By(fmt.Sprintf("delete service (%s)", service.Name))
		err := f.ClientSet.CoreV1().Services(ns.Name).Delete(service.Name, &metav1.DeleteOptions{})
		Expect(err).NotTo(HaveOccurred())
	}

	By(fmt.Sprintf("delete deployment (%s)", r.Deployment.Name))
	err := f.ClientSet.AppsV1().Deployments(ns.Name).Delete(r.Deployment.Name, &metav1.DeleteOptions{})
	Expect(err).NotTo(HaveOccurred())

	By(fmt.Sprintf("wait delete deployment (%s)", r.Deployment.Name))
	err = f.ResourceManager.WaitDeploymentDeleted(ctx, r.Deployment)
	Expect(err).NotTo(HaveOccurred())
}

// PatchSpec is for Kubernetes patching
type PatchSpec struct {
	Op    string `json:"op"`
	Path  string `json:"path"`
	Value int32  `json:"value"`
}

// ExpectDeploymentScaleSuccessful expects a deployment to scale sucessfully
func (r *Resources) ExpectDeploymentScaleSuccessful(ctx context.Context, f *framework.Framework, ns *corev1.Namespace, replicas int32) {
	// TODO: change to scale when client-go is updated
	// scale, err := f.ClientSet.AppsV1().Deployments(ns.Name).GetScale(r.Deployment.Name, &metav1.GetOptions{})
	// scale.Spec.Replicas = replicas
	// scale, err = f.ClientSet.AppsV1().Deployments(ns.Name).UpdateScale(r.Deployment.Name, &scale)
	By(fmt.Sprintf("scale deployment (%s)", r.Deployment.Name))
	patch := []PatchSpec{
		{
			Op:    "replace",
			Path:  "/spec/replicas",
			Value: replicas,
		},
	}
	patchBytes, err := json.Marshal(patch)

	dp, err := f.ClientSet.AppsV1().Deployments(ns.Name).Patch(r.Deployment.Name, types.JSONPatchType, patchBytes)
	Expect(err).NotTo(HaveOccurred())

	By(fmt.Sprintf("wait deployment (%s)", r.Deployment.Name))
	_, err = f.ResourceManager.WaitDeploymentReady(ctx, dp)
	Expect(err).NotTo(HaveOccurred())
}

// ExpectDaemonsetUpdateSuccessful expects updating a daemonset to be successful
func (r *Resources) ExpectDaemonsetUpdateSuccessful(ctx context.Context, f *framework.Framework, ns *corev1.Namespace) {
	By(fmt.Sprintf("update daemonset (%s)", r.Daemonset.Name))
	ds, err := f.ClientSet.AppsV1().DaemonSets(ns.Name).Update(r.Daemonset)
	Expect(err).NotTo(HaveOccurred())

	By(fmt.Sprintf("wait daemonset (%s)", r.Daemonset.Name))
	_, err = f.ResourceManager.WaitDaemonSetReady(ctx, ds)
	Expect(err).NotTo(HaveOccurred())
}

// ExpectServicesSuccessful expects service creation to be successful
func (r *Resources) ExpectServicesSuccessful(ctx context.Context, f *framework.Framework, ns *corev1.Namespace, replicas int) {
	for _, service := range r.Services {
		By(fmt.Sprintf("create service (%s)", service.Name))
		svc, err := f.ClientSet.CoreV1().Services(ns.Name).Create(service)
		Expect(err).NotTo(HaveOccurred())
		By(fmt.Sprintf("wait service (%s)", service.Name))
		_, err = f.ResourceManager.WaitServiceHasEndpointsNum(ctx, svc, replicas)
		Expect(err).NotTo(HaveOccurred())
	}
}
