package resource

import (
	"context"

	"github.com/aws/amazon-vpc-cni-k8s/test/e2e/framework/utils"
	appsv1 "k8s.io/api/apps/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/client-go/kubernetes"
)

type DeploymentManager struct {
	cs kubernetes.Interface
}

func NewDeploymentManager(cs kubernetes.Interface) *DeploymentManager {
	return &DeploymentManager{
		cs: cs,
	}
}

// From DeploymentComplete in k8s.io/kubernetes/pkg/controller/deployment/util/deployment_util.go
func (m *DeploymentManager) WaitDeploymentReady(ctx context.Context, dp *appsv1.Deployment) (*appsv1.Deployment, error) {
	var (
		observedDP *appsv1.Deployment
		err        error
	)
	// start := time.Now()

	return observedDP, wait.PollImmediateUntil(utils.PollIntervalShort, func() (bool, error) {
		observedDP, err = m.cs.AppsV1().Deployments(dp.Namespace).Get(dp.Name, metav1.GetOptions{})
		if err != nil {
			return false, err
		}

		// log.Debugf("%d / %d pods ready in namespace '%s' in deployment '%s' (%d seconds elapsed)",
		// 	observedDP.Status.AvailableReplicas, observedDP.Status.Replicas, dp.Namespace,
		// 	observedDP.ObjectMeta.Name, int(time.Since(start).Seconds()))

		if observedDP.Status.UpdatedReplicas == (*dp.Spec.Replicas) &&
			observedDP.Status.Replicas == (*dp.Spec.Replicas) &&
			observedDP.Status.AvailableReplicas == (*dp.Spec.Replicas) &&
			observedDP.Status.ObservedGeneration >= dp.Generation {
			return true, nil
		}
		return false, nil
	}, ctx.Done())
}

func (m *DeploymentManager) WaitDeploymentDeleted(ctx context.Context, dp *appsv1.Deployment) error {
	var (
		err error
	)
	return wait.PollImmediateUntil(utils.PollIntervalShort, func() (bool, error) {
		_, err = m.cs.AppsV1().Deployments(dp.Namespace).Get(dp.Name, metav1.GetOptions{})
		if err != nil {
			if serr, ok := err.(*errors.StatusError); ok {
				switch serr.ErrStatus.Reason {
				case "NotFound":
					return true, nil
				default:
					return false, err
				}
			}
			return false, err
		}
		return false, nil
	}, ctx.Done())
}
