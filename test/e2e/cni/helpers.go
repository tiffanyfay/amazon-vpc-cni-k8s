// TODO figure out if this name should change or where these functions should go
package cni

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"net/http"
	"time"

	"github.com/aws/amazon-vpc-cni-k8s/ipamd/datastore"
	"github.com/tiffanyfay/aws-k8s-test-framework/test/e2e/framework"
	"github.com/tiffanyfay/aws-k8s-test-framework/test/e2e/resources"

	"github.com/aws/amazon-vpc-cni-k8s/pkg/awsutils"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/awserr"
	"github.com/aws/aws-sdk-go/service/autoscaling"
	"github.com/aws/aws-sdk-go/service/ec2"
	log "github.com/cihub/seelog"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// UpdateAWSNodeEnvs updates the aws-node daemonset's WARM_IP_TARGET, WARM_ENI_TARGET, and MAX_ENI and waits for
// the daemonset to be updated
func UpdateAWSNodeEnvs(ctx context.Context, f *framework.Framework, warmIPTarget, warmENITarget, maxENI string) {
	// Get aws-node daemonset
	ks := &corev1.Namespace{ObjectMeta: metav1.ObjectMeta{Name: "kube-system"}}
	ds, err := f.ClientSet.AppsV1().DaemonSets(ks.Name).Get("aws-node", metav1.GetOptions{})
	Expect(err).ShouldNot(HaveOccurred())

	// Update env vars
	for i, envar := range ds.Spec.Template.Spec.Containers[0].Env {
		if envar.Name == "WARM_IP_TARGET" {
			ds.Spec.Template.Spec.Containers[0].Env[i].Value = warmIPTarget
		} else if envar.Name == "WARM_ENI_TARGET" {
			ds.Spec.Template.Spec.Containers[0].Env[i].Value = warmENITarget
		} else if envar.Name == "MAX_ENI" {
			ds.Spec.Template.Spec.Containers[0].Env[i].Value = maxENI
		}
	}
	// Update aws-node daemonset
	resource := &resources.Resources{
		Daemonset: ds,
	}
	resource.ExpectDaemonsetUpdateSuccessful(ctx, f, ks)
}

// TestENIInfo checks if the ENIInfo values are as expected
func TestENIInfo(ctx context.Context, f *framework.Framework, internalIP string, expectedENICount int, expectedIPCount int) {
	// TODO: Future upgrade can get the port from aws-node's INTROSPECTION_BIND_ADDRESS

	// Sleep needed because metrics endpoint takes time to update
	time.Sleep(time.Second * 10)

	port := "61679"

	svc := &corev1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "aws-node",
			Namespace: "kube-system",
		},
	}

	_, err := f.ResourceManager.WaitServiceHasEndpointIP(ctx, svc, internalIP)
	Expect(err).ShouldNot(HaveOccurred())

	enisPath := fmt.Sprintf("http://%s:%s/v1/enis", internalIP, port)
	resp, err := http.Get(enisPath) // TODO add retry/wait logic
	Expect(err).ShouldNot(HaveOccurred())
	defer resp.Body.Close()

	body, err := ioutil.ReadAll(resp.Body)
	Expect(err).ShouldNot(HaveOccurred())

	var eniInfos datastore.ENIInfos
	json.Unmarshal(body, &eniInfos)
	log.Debugf("%+v", eniInfos)
	log.Debugf("Expected ENI count %d", expectedENICount)

	// TODO check EC2 instance
	By("checking number of ENIs")
	Expect(len(eniInfos.ENIIPPools)).To(Equal(expectedENICount))

	By("checking number of IPs")
	for k, v := range eniInfos.ENIIPPools {
		log.Debugf("Checking number of IPs for %s", k)
		Expect(len(v.IPv4Addresses)).To(Equal(expectedIPCount))
	}
}

// TODO make it take in the ns and pod name
// GetTesterPodNodeName gets the node name in which the pod runs on
func GetTesterPodNodeName(f *framework.Framework, nsName string, podName string) (string, error) {
	testerPod, err := f.ClientSet.CoreV1().Pods(nsName).Get(podName, metav1.GetOptions{})
	return testerPod.Spec.NodeName, err
}

// TODO GetTestNodes
func GetTestNodes(f *framework.Framework, testerNodeName string) ([]corev1.Node, error) {
	var testNodes []corev1.Node

	nodesList, err := f.ClientSet.CoreV1().Nodes().List(metav1.ListOptions{})
	if err != nil {
		return nil, err
	}
	if len(nodesList.Items) == 0 {
		return nil, errors.New("No nodes found")
	}

	for _, node := range nodesList.Items {
		if testerNodeName != node.Name {
			log.Debugf("Found test node (%s)", node.Name)
			testNodes = append(testNodes, node)
		}
	}
	return testNodes, nil
}

// ReplaceASGInstances terminates instances for given nodes, waits for new instances to be
// ready in their autoscaling groups, and waits for the new nodes to be ready
func ReplaceASGInstances(ctx context.Context, f *framework.Framework, nodes []corev1.Node) error {
	var asgs []*string
	var nodeNames []*string
	var instanceIDsTerminate []*string
	var instanceIDs []*string

	for _, node := range nodes {
		nodeName := node.Name
		nodeNames = append(nodeNames, &nodeName)
	}

	// Get instance IDs
	filterName := "private-dns-name"
	describeInstancesInput := &ec2.DescribeInstancesInput{
		Filters: []*ec2.Filter{
			{
				Name:   &filterName,
				Values: nodeNames,
			},
		},
	}
	instancesToTerminate, err := f.Cloud.EC2().DescribeInstancesAsList(aws.BackgroundContext(), describeInstancesInput)
	if err != nil {
		if aerr, ok := err.(awserr.Error); ok {
			switch aerr.Code() {
			default:
				log.Debug(aerr)
			}
		} else {
			log.Debug(err)
		}
		return err
	}
	if len(instancesToTerminate) == 0 {
		return errors.New("No instances found")
	}
	for i, instance := range instancesToTerminate {
		log.Debugf("Terminating instance %d/%d (name: %v, id: %v)", i+1, len(instancesToTerminate), *(instance.PrivateDnsName), *(instance.InstanceId))
		instanceIDsTerminate = append(instanceIDsTerminate, instance.InstanceId)
	}
	// Terminate instances
	for _, instanceID := range instanceIDsTerminate {
		terminateInstanceInASGInput := &autoscaling.TerminateInstanceInAutoScalingGroupInput{
			InstanceId:                     aws.String(*instanceID),
			ShouldDecrementDesiredCapacity: aws.Bool(false),
		}
		result, err := f.Cloud.AutoScaling().TerminateInstanceInAutoScalingGroup(terminateInstanceInASGInput)
		if err != nil {
			if aerr, ok := err.(awserr.Error); ok {
				switch aerr.Code() {
				case autoscaling.ErrCodeScalingActivityInProgressFault:
					log.Debug(autoscaling.ErrCodeScalingActivityInProgressFault, aerr.Error())
				case autoscaling.ErrCodeResourceContentionFault:
					log.Debug(autoscaling.ErrCodeResourceContentionFault, aerr.Error())
				default:
					log.Debug(aerr.Error())
				}
			} else {
				// Print the error, cast err to awserr.Error to get the Code and
				// Message from an error.
				log.Debug(err.Error())
			}
			return err
		}
		asgs = append(asgs, result.Activity.AutoScalingGroupName)
	}

	time.Sleep(time.Second * 2)

	// Wait for ASGs to be in service
	// Need to make sure that min == desired
	describeASGsInput := &autoscaling.DescribeAutoScalingGroupsInput{
		AutoScalingGroupNames: asgs,
	}

	By("wait ASG instances are ready")
	err = f.Cloud.AutoScaling().WaitUntilAutoScalingGroupInService(aws.BackgroundContext(), describeASGsInput)

	// Get new instance IDs
	instances, err := f.Cloud.AutoScaling().DescribeInServiceAutoScalingGroupInstancesAsList(aws.BackgroundContext(), describeASGsInput)
	if err != nil {
		return err
	}

	By("wait nodes ready")
	for i, instance := range instances {
		log.Debugf("Instance %d/%d (id: %s) is in service", i+1, len(instances), *(instance.InstanceId))
		instanceIDs = append(instanceIDs, instance.InstanceId)
	}
	describeInstancesInput = &ec2.DescribeInstancesInput{
		InstanceIds: instanceIDs,
	}
	instancesList, err := f.Cloud.EC2().DescribeInstancesAsList(aws.BackgroundContext(), describeInstancesInput)
	if err != nil {
		if aerr, ok := err.(awserr.Error); ok {
			switch aerr.Code() {
			default:
				log.Debug(aerr)
			}
		} else {
			log.Debug(err)
		}
	}

	// Wait until nodes exists and are ready
	for i, instance := range instancesList {
		nodeName := instance.PrivateDnsName
		log.Debugf("Wait until node %d/%d (%s) exists", i+1, len(instancesList), *nodeName)
		node := &corev1.Node{ObjectMeta: metav1.ObjectMeta{Name: *nodeName}}
		node, err = f.ResourceManager.WaitNodeExists(ctx, node)
		if err != nil {
			return err
		}
		log.Debugf("Wait until node %d/%d (%s) ready", i+1, len(instancesList), *nodeName)
		_, err = f.ResourceManager.WaitNodeReady(ctx, node)
		if err != nil {
			return err
		}
	}
	return nil
}

// Get instance ENI and IP limits
func GetInstanceLimits(f *framework.Framework, nodeName string) (int, int, error) {
	filterName := "private-dns-name"
	describeInstancesInput := &ec2.DescribeInstancesInput{
		Filters: []*ec2.Filter{
			{
				Name:   &filterName,
				Values: []*string{&nodeName},
			},
		},
	}
	instance, err := f.Cloud.EC2().DescribeInstances(describeInstancesInput)
	if err != nil {
		return 0, 0, err
	}
	if len(instance.Reservations) < 1 {
		return 0, 0, errors.New("No instance reservations found")
	}
	if len(instance.Reservations[0].Instances) < 1 {
		return 0, 0, errors.New("No instances found")
	}
	instanceType := *(instance.Reservations[0].Instances[0].InstanceType)

	return awsutils.InstanceENIsAvailable[instanceType],
		awsutils.InstanceIPsAvailable[instanceType] - 1, nil
}

// GetNodeInternalIP gets a node's internal IP address
func GetNodeInternalIP(node corev1.Node) (string, error) {
	if len(node.Status.Addresses) == 0 {
		return "", fmt.Errorf("No addresses found for node (%s)", node.Name)
	}

	var internalIP string
	for _, address := range node.Status.Addresses {
		if address.Type == corev1.NodeInternalIP {
			internalIP = address.Address
		}
	}
	return internalIP, nil
}
