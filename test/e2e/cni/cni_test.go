package cni_test

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"net/http"
	"time"

	"github.com/aws/amazon-vpc-cni-k8s/ipamd/datastore"
	"github.com/aws/amazon-vpc-cni-k8s/test/e2e/framework"
	"github.com/aws/amazon-vpc-cni-k8s/test/e2e/framework/utils"
	"github.com/aws/amazon-vpc-cni-k8s/test/e2e/resources"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/awserr"
	"github.com/aws/aws-sdk-go/service/autoscaling"
	"github.com/aws/aws-sdk-go/service/ec2"
	log "github.com/cihub/seelog"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	promv1 "github.com/prometheus/client_golang/api/prometheus/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// Timeout for waiting events in seconds// const TIMEOUT = 60
var (
	err      error
	limit    float32
	ctx      context.Context
	testTime time.Time

	f                   *framework.Framework
	prom                *resources.Prom
	awsNodeSvc          *resources.Resources
	promResources       *resources.Resources
	testpodResources    *resources.Resources
	nginxResourcesGroup *resources.ResourcesGroup

	promAPI promv1.API
	ns      *corev1.Namespace
	// nodes   []corev1.Node
)

// func setup() {
// 	limit = 0.1

// 	f, err = framework.NewFastFramework()
// 	Expect(err).NotTo(HaveOccurred())

// 	ns = &corev1.Namespace{ObjectMeta: metav1.ObjectMeta{Name: "cni-test"}}
// 	ctx = context.Background()

// 	// TODO should we make sure they are on each node?
// 	// testpodResources = resources.NewTestpodResources(ns.Name, 6)
// 	// testpodResources.ExpectDeploySuccessful(ctx, f, ns)
// }

func expectTestpodPromMetricsPass() {
	It("should get number of events received", func() {
		// TODO: set it for some # of expected requests?
		receivedTotal, err := prom.Query("cni_test_received_total", testTime)
		log.Infof("receivedTotal %v", receivedTotal)
		Expect(err).NotTo(HaveOccurred())
		Expect(receivedTotal).NotTo(BeNil())
	})
	It("should get dnsRequestFailurePercent below limit", func() {
		dnsRequestFailurePercent, err := prom.QueryPercent("cni_test_dns_request_total",
			"cni_test_dns_request_failure", testTime)
		Expect(err).NotTo(HaveOccurred())
		Expect(dnsRequestFailurePercent).NotTo(BeNil())
		log.Infof("dnsRequestFailurePercent %v", dnsRequestFailurePercent)
		Expect(dnsRequestFailurePercent).To(BeNumerically("<", limit))
	})
	It("should get externalHTTPRequestsFailurePercent below limit", func() {
		externalHTTPRequestsFailurePercent, err := prom.QueryPercent("cni_test_external_http_request_total",
			"cni_test_external_http_request_failure", testTime)
		Expect(err).NotTo(HaveOccurred())
		Expect(externalHTTPRequestsFailurePercent).NotTo(BeNil())
		Expect(externalHTTPRequestsFailurePercent).To(BeNumerically("<", limit))
	})
	It("should get svcClusterIPRequestFailurePercent below limit QueryPercent", func() {
		svcClusterIPRequestFailurePercent, err := prom.QueryPercent("cni_test_cluster_ip_request_total",
			"cni_test_cluster_ip_request_failure", testTime)
		Expect(err).NotTo(HaveOccurred())
		Expect(svcClusterIPRequestFailurePercent).NotTo(BeNil())
		Expect(svcClusterIPRequestFailurePercent).To(BeNumerically("<", limit))
	})
	It("should get svcPodIPRequestsFailurePercent below limit", func() {
		svcPodIPRequestsFailurePercent, err := prom.QueryPercent("cni_test_external_http_request_total",
			"cni_test_external_http_request_failure", testTime)
		Expect(err).NotTo(HaveOccurred())
		Expect(svcPodIPRequestsFailurePercent).NotTo(BeNil())
		Expect(svcPodIPRequestsFailurePercent).To(BeNumerically("<", limit))
	})
}

func expectAWSNodePromMetricsPass() {

}

// TODO make it take in the ns and pod name
// getTesterNodeName gets the node name in which the cni-e2e test runs on
func getTesterNodeName() (string, error) {
	testerPod, err := f.ClientSet.CoreV1().Pods("cni-test").Get("cni-e2e", metav1.GetOptions{})
	return testerPod.Spec.NodeName, err
}

func getTestNodes() ([]corev1.Node, error) {
	var testNodes []corev1.Node

	testerNode, err := getTesterNodeName()
	if err != nil {
		return nil, err
	}

	time.Sleep(time.Second * 10)
	nodesList, err := f.ClientSet.CoreV1().Nodes().List(metav1.ListOptions{})
	if err != nil {
		return nil, err
	}
	if len(nodesList.Items) == 0 {
		return nil, errors.New("No nodes found")
	}

	for i, node := range nodesList.Items {
		if testerNode == node.Name {
			log.Debugf("tester node %d/%d (%s)", i+1, len(nodesList.Items), node.Name)
			continue
		}
		log.Debugf("test node %d/%d  (%s)", i+1, len(nodesList.Items), node.Name)
		testNodes = append(testNodes, node)
	}
	return testNodes, nil
}

// replaceASGInstances terminates instances for given nodes, waits for new instances to be
// ready in their autoscaling groups, and waits for the new nodes to be ready
func replaceASGInstances(nodes []corev1.Node) error {
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
		log.Debugf("terminating instance %d/%d (name: %v, id: %v)", i+1, len(instancesToTerminate), *(instance.PrivateDnsName), *(instance.InstanceId))
		instanceIDsTerminate = append(instanceIDsTerminate, instance.InstanceId)
	}
	// Terminate instances
	for _, instanceID := range instanceIDsTerminate {
		terminateInstanceInASGInput := &autoscaling.TerminateInstanceInAutoScalingGroupInput{
			InstanceId:                     aws.String(*instanceID),
			ShouldDecrementDesiredCapacity: aws.Bool(false),
		}
		result, err := f.Cloud.Autoscaling().TerminateInstanceInAutoScalingGroup(terminateInstanceInASGInput)
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

	By("wait until instances are terminated")
	for _, instanceID := range instanceIDsTerminate {
		log.Debugf("waiting until instance (%s) is terminated", *instanceID)
		describeInstancesInput = &ec2.DescribeInstancesInput{
			InstanceIds: []*string{instanceID},
		}

		// Wait for instances to be terminated
		err = f.Cloud.EC2().WaitUntilInstanceTerminated(describeInstancesInput)
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
	}

	// Wait until instance is not in ASG

	// Wait for ASGs to be ready
	// Need to make sure that min == desired
	describeASGsInput := &autoscaling.DescribeAutoScalingGroupsInput{
		AutoScalingGroupNames: asgs,
	}

	// Get new instance IDs
	instances, err := f.Cloud.Autoscaling().DescribeAutoScalingGroupInstancesAsList(ctx, describeASGsInput)
	if err != nil {
		return err
	}

	// var done bool
	// for !done {
	// 	instances, err = f.Cloud.Autoscaling().DescribeAutoScalingGroupInstancesAsList(ctx, describeASGsInput)
	// 	if err != nil {
	// 		return err
	// 	}
	// 	log.Debugf("ni: number of instances %d", len(instances))
	// 	for _, instance := range instances {
	// 		log.Debugf("ni: instance state (%s) status %s", *(instance.InstanceId), *(instance.LifecycleState))
	// 		if *(instance.LifecycleState) == autoscaling.LifecycleStateTerminating {
	// 			log.Debugf("ni: instance terminating (%s) status %s", *(instance.InstanceId), *(instance.LifecycleState))
	// 			done = true
	// 		}
	// 	}

	// }

	By("wait ASG instances are ready")
	asgout, err := f.Cloud.Autoscaling().DescribeAutoScalingGroups(describeASGsInput)
	if err != nil {
		return err
	}
	min := asgout.AutoScalingGroups[0].MinSize

	var done2 bool
	for !done2 {
		instances, err = f.Cloud.Autoscaling().DescribeAutoScalingGroupInstancesAsList(ctx, describeASGsInput)
		if err != nil {
			return err
		}
		var count int64
		log.Debugf("in: number of instances %d", len(instances))
		for _, instance := range instances {
			log.Debugf("in: instance state (%s) status %s", *(instance.InstanceId), *(instance.LifecycleState))
			if *(instance.LifecycleState) == autoscaling.LifecycleStateInService {
				count++
			}
			if count == *min {
				done2 = true
			}
		}
		time.Sleep(time.Second * 10)
	}

	// Get new instance IDs
	instances, err = f.Cloud.Autoscaling().DescribeInServiceAutoScalingGroupInstancesAsList(ctx, describeASGsInput)
	if err != nil {
		return err
	}

	// Wait for nodes to be ready
	By("wait nodes ready")
	for i, instance := range instances {
		log.Debugf("instance %d/%d (id: %s) is in service", i+1, len(instances), *(instance.InstanceId))
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

	for i, instance := range instancesList {
		// Wait until node exists and is ready
		nodeName := instance.PrivateDnsName
		log.Debugf("wait until node %d/%d (%s) exists", i+1, len(instancesList), *nodeName)
		node := &corev1.Node{ObjectMeta: metav1.ObjectMeta{Name: *nodeName}}
		node, err = f.ResourceManager.WaitNodeExists(ctx, node)
		if err != nil {
			return err
		}
		log.Infof("wait until node (%s) is ready", *nodeName)
		_, err = f.ResourceManager.WaitNodeReady(ctx, node)
		if err != nil {
			return err
		}
	}
	return nil
}

func expectUpdateAWSNodeSuccessful(warmIPTarget, warmENITarget, maxENI string) {
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

// TODO
func testENIInfo(nodes []corev1.Node, expectedENICount int, expectedIPCount int) {
	for _, node := range nodes {
		Expect(node.Status).NotTo(BeNil())
		Expect(node.Status.Addresses).NotTo(BeNil())

		// Get node port IP for metrics
		var internalIP string
		for _, address := range node.Status.Addresses {
			if address.Type == corev1.NodeInternalIP {
				internalIP = address.Address
			}
		}
		port := "61679"

		log.Debugf("Node (%s) has internal IP %s", node.Name, internalIP)
		svc := &corev1.Service{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "aws-node",
				Namespace: "kube-system",
			},
		}

		_, err = f.ResourceManager.WaitServiceHasEndpointIP(ctx, svc, internalIP)
		Expect(err).ShouldNot(HaveOccurred())

		enisPath := fmt.Sprintf("http://%s:%s/v1/enis", internalIP, port)
		resp, err := http.Get(enisPath)
		Expect(err).ShouldNot(HaveOccurred())
		defer resp.Body.Close()

		body, err := ioutil.ReadAll(resp.Body)
		Expect(err).ShouldNot(HaveOccurred())

		var eniInfos datastore.ENIInfos
		json.Unmarshal(body, &eniInfos)
		log.Debugf("%+v", eniInfos)
		log.Debugf("Expected ENI count %d", expectedENICount)

		By("checking number of ENIs")
		// Check number of ENIs
		Expect(len(eniInfos.ENIIPPools)).To(Equal(expectedENICount))

		By("checking number of IPs")
		// Check number of IPs per ENI
		for k, v := range eniInfos.ENIIPPools {
			log.Debugf("checking number of IPs for %s", k)
			Expect(len(v.IPv4Addresses)).To(Equal(expectedIPCount))
		}
	}
}

// TODO move to cni_suite_test.go
// var _ = BeforeSuite(setup)
func createNginxResources(nodes []corev1.Node) {
	// Create NGINX resources
	var nginxResources []*resources.Resources
	// testpodResources.ExpectDeploymentScaleSuccessful(ctx, f, ns, 23)
	// Create NGINX resources
	Expect(len(nodes)).To(BeNumerically(">", 0))
	for i, node := range nodes {
		log.Infof("creating nginx for node %d: %v", i+1, node.Name)
		nginxResource := resources.NewNginxResources(ns.Name, node.Name, 0)
		nginxResource.ExpectDeploySuccessful(ctx, f, ns)
		nginxResources = append(nginxResources, nginxResource)
	}

	nginxResourcesGroup = &resources.ResourcesGroup{
		ResourcesGroup: nginxResources,
	}
}

var _ = Describe("Testing 1 node", func() {
	f = framework.New()

	ns = &corev1.Namespace{ObjectMeta: metav1.ObjectMeta{Name: "cni-test"}}
	ctx = context.Background()

	kubeSystem := &corev1.Namespace{ObjectMeta: metav1.ObjectMeta{Name: "kube-system"}}

	BeforeEach(func() {
		// Create prometheus resources
		// testerNode, err := getTesterNodeName()
		// promResources = resources.NewPromResources(ns.Name, testerNode, 1)
		// promResources.ExpectDeploySuccessful(ctx, f, ns)
		// promAPI, err = resources.NewPromAPI(f, ns)
		// Expect(err).NotTo(HaveOccurred())
		// time.Sleep(time.Second * 5)
		// prom = &resources.Prom{API: promAPI}

	})

	// time.Sleep(time.Second * 5)
	// testTime := time.Now()

	// Context("With CNI testpods and prometheus metrics", expectTestpodPromMetricsPass)

	// Context("With IPAMD and prometheus metrics", func() {
	// 	It("awsCNIAWSAPIErrorCount should be 0", func() {
	// 		QueryPercent, err := prom.Query("awscni_aws_api_error_count", testTime)
	// 		Expect(err).NotTo(HaveOccurred())
	// 		Expect(QueryPercent).NotTo(BeNil())
	// 		Expect(QueryPercent).To(BeNumerically("<=", 5))
	// 	})
	// })

	It("Should pass with WARM_IP_TARGET=0 (default), WARM_ENI_TARGET=1 (default), and MAX_ENI=-1(default)", func() {
		expectUpdateAWSNodeSuccessful("0", "1", "-1")
		nodes, err := getTestNodes()
		Expect(err).ShouldNot(HaveOccurred())
		err = replaceASGInstances(nodes)
		Expect(err).ShouldNot(HaveOccurred())
		awsNodeDS, err := f.ClientSet.AppsV1().DaemonSets(kubeSystem.Name).Get("aws-node", metav1.GetOptions{})
		Expect(err).ShouldNot(HaveOccurred())
		awsNodeS, err := f.ClientSet.CoreV1().Services(kubeSystem.Name).Get("aws-node", metav1.GetOptions{})
		Expect(err).ShouldNot(HaveOccurred())
		f.ResourceManager.WaitDaemonSetReady(ctx, awsNodeDS)

		f.ResourceManager.WaitServiceHasEndpointsNum(ctx, awsNodeS, int(awsNodeDS.Status.NumberReady))
		nodes, err = getTestNodes()
		Expect(err).ShouldNot(HaveOccurred())
		createNginxResources(nodes)

		By("scaling testpod up to 25 pods")
		nginxResourcesGroup.ExpectDeploymentScaleSuccessful(ctx, f, ns, 25)
		time.Sleep(time.Second * 5)
		testENIInfo(nodes, 3, 14)

		By("scaling testpod up to 30 pods") // look into why it isn't changing at 26
		nginxResourcesGroup.ExpectDeploymentScaleSuccessful(ctx, f, ns, 30)
		time.Sleep(time.Second * 15) // TODO handle this because the ENI info is slower to load
		testENIInfo(nodes, 4, 20)
	})

	// It("Should pass with WARM_IP_TARGET=0 (default), WARM_ENI_TARGET=0, and MAX_ENI=-1(default)", func() {
	// 	expectUpdateAWSNodeSuccessful("0", "0", "-1")
	// 	nodes, err := getTestNodes()
	// 	Expect(err).ShouldNot(HaveOccurred())
	// 	err = replaceASGInstances(nodes)
	// 	Expect(err).ShouldNot(HaveOccurred())
	// 	awsNodeDS, err := f.ClientSet.AppsV1().DaemonSets(kubeSystem.Name).Get("aws-node", metav1.GetOptions{})
	// 	Expect(err).ShouldNot(HaveOccurred())
	// 	awsNodeS, err := f.ClientSet.CoreV1().Services(kubeSystem.Name).Get("aws-node", metav1.GetOptions{})
	// 	Expect(err).ShouldNot(HaveOccurred())
	// 	f.ResourceManager.WaitDaemonSetReady(ctx, awsNodeDS)

	// 	f.ResourceManager.WaitServiceHasEndpointsNum(ctx, awsNodeS, int(awsNodeDS.Status.NumberReady))
	// 	nodes, err = getTestNodes()
	// 	Expect(err).ShouldNot(HaveOccurred())
	// 	createNginxResources(nodes)

	// 	By("scaling testpod up to 25 pods")
	// 	nginxResourcesGroup.ExpectDeploymentScaleSuccessful(ctx, f, ns, 25)
	// 	time.Sleep(time.Second * 6)
	// 	testENIInfo(nodes, 2, 14)

	// 	By("scaling testpod up to 26 pods")
	// 	nginxResourcesGroup.ExpectDeploymentScaleSuccessful(ctx, f, ns, 26)
	// 	time.Sleep(time.Second * 6) // TODO handle this because the ENI info is slower to load
	// 	testENIInfo(nodes, 3, 14)
	// })

	AfterEach(func() {
		// Replace nodes

		// promResources.ExpectCleanupSuccessful(ctx, f, ns)
		nginxResourcesGroup.ExpectCleanupSuccessful(ctx, f, ns)
		// awsNodeSvc.ExpectCleanupSuccessful(ctx, f, kubeSystem)
	})
})

var _ = SynchronizedAfterSuite(func() {
	// Run on all Ginkgo nodes
	utils.Logf("Running AfterSuite actions on all nodes")
	framework.RunCleanupActions()
}, func() {
	Expect(f).NotTo(BeNil())
	// nginxResourcesGroup.ExpectCleanupSuccessful(ctx, f, ns)

	// testpodResources.ExpectDeploySuccessful(ctx, f, ns)
})
