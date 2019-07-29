package cloud

import (
	"context"

	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/autoscaling"
	"github.com/aws/aws-sdk-go/service/autoscaling/autoscalingiface"
	log "github.com/cihub/seelog"
)

// Autoscaling is an wrapper around the original AutoscalingAPI with additional convenient APIs.
type Autoscaling interface {
	autoscalingiface.AutoScalingAPI
	DescribeAutoScalingGroupInstancesAsList(ctx context.Context, input *autoscaling.DescribeAutoScalingGroupsInput) ([]*autoscaling.Instance, error)
	DescribeInServiceAutoScalingGroupInstancesAsList(ctx context.Context, input *autoscaling.DescribeAutoScalingGroupsInput) ([]*autoscaling.Instance, error)
}

// NewAutoscaling creates a new autoscaling session
func NewAutoscaling(session *session.Session) Autoscaling {
	return &defaultAutoscaling{
		autoscaling.New(session),
	}
}

var _ Autoscaling = (*defaultAutoscaling)(nil)

type defaultAutoscaling struct {
	autoscalingiface.AutoScalingAPI
}

func (c *defaultAutoscaling) DescribeAutoScalingGroupInstancesAsList(ctx context.Context, input *autoscaling.DescribeAutoScalingGroupsInput) ([]*autoscaling.Instance, error) {
	var result []*autoscaling.Instance
	if err := c.DescribeAutoScalingGroupsPagesWithContext(ctx, input, func(output *autoscaling.DescribeAutoScalingGroupsOutput, _ bool) bool {
		for _, item := range output.AutoScalingGroups {
			result = append(result, item.Instances...)
		}
		return true
	}); err != nil {
		return nil, err
	}
	return result, nil
}

func (c *defaultAutoscaling) DescribeInServiceAutoScalingGroupInstancesAsList(ctx context.Context, input *autoscaling.DescribeAutoScalingGroupsInput) ([]*autoscaling.Instance, error) {
	var instances []*autoscaling.Instance
	var result []*autoscaling.Instance
	if err := c.DescribeAutoScalingGroupsPagesWithContext(ctx, input, func(output *autoscaling.DescribeAutoScalingGroupsOutput, _ bool) bool {
		for _, item := range output.AutoScalingGroups {
			instances = append(instances, item.Instances...)
		}
		for _, instance := range instances {
			log.Debugf("all lifecyle (%s) status (%s)", *(instance.InstanceId), *(instance.LifecycleState))
			if *(instance.LifecycleState) == autoscaling.LifecycleStateInService {
				result = append(result, instance)
			}
		}
		return true
	}); err != nil {
		return nil, err
	}
	return result, nil
}
