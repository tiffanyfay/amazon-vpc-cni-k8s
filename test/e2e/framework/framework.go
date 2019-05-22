package framework

import (
	"context"

	"github.com/aws/amazon-vpc-cni-k8s/pkg/awsutils"

	// "github.com/kubernetes-sigs/aws-alb-ingress-controller/internal/aws"

	// "github.com/kubernetes-sigs/aws-alb-ingress-controller/internal/ingress/metric"
	"github.com/aws/amazon-vpc-cni-k8s/test/e2e/framework/resource"
	"github.com/aws/amazon-vpc-cni-k8s/test/e2e/framework/utils"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	clientset "k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
)

// TODO(@M00nF1sh): migrate to use k8s test framework when it's isolated without pulling in all dependencies.
// This is an simplified version of k8s test framework, since some dependencies don't work under go module :(.
// Framework supports common operations used by e2e tests; it will keep a client & a namespace for you.
type Framework struct {
	ClientSet clientset.Interface
	// Cloud           cloud.Cloud
	AWSClient       awsutils.APIs
	ResourceManager *resource.Manager
	Config          *rest.Config // TODO delete me

	Options Options

	// To make sure that this framework cleans up after itself, no matter what,
	// we install a Cleanup action before each test and clear it after.  If we
	// should abort, the AfterSuite hook should run all Cleanup actions.
	cleanupHandle CleanupActionHandle
}

// New makes a new framework and sets up a BeforeEach/AfterEach for you.
func New() *Framework {
	f := &Framework{
		Options: globalOptions,
	}

	BeforeEach(f.BeforeEach)
	AfterEach(f.AfterEach)

	return f
}

func (f *Framework) BeforeEach() {
	// The fact that we need this feels like a bug in ginkgo.
	// https://github.com/onsi/ginkgo/issues/222
	if f.ClientSet == nil {
		var err error
		restCfg, err := f.buildRestConfig()
		Expect(err).NotTo(HaveOccurred())
		f.Config = restCfg // TODO delete me
		f.ClientSet, err = clientset.NewForConfig(restCfg)
		Expect(err).NotTo(HaveOccurred())
	}
	// if f.Cloud == nil {
	// 	// reg := prometheus.NewRegistry()
	// 	// mc, _ := metric.NewCollector(reg, "alb")
	// 	var err error
	// 	f.Cloud, err = cloud.New(aws.CloudConfig{
	// 		Region: f.Options.AWSRegion,
	// 		VpcID:  f.Options.AWSVPCID},
	// 		f.Options.ClusterName, mc)
	// 	gomega.Expect(err).NotTo(gomega.HaveOccurred())
	// }
	if f.AWSClient == nil {
		// reg := prometheus.NewRegistry()
		// mc, _ := metric.NewCollector(reg, "alb")
		var err error
		f.AWSClient, err = awsutils.New()
		Expect(err).NotTo(HaveOccurred())
	}

	f.ResourceManager = resource.NewManager(f.ClientSet)
	f.cleanupHandle = AddCleanupAction(f.cleanupAction())
}

func (f *Framework) AfterEach() {
	RemoveCleanupAction(f.cleanupHandle)

	f.cleanupAction()()
}

func (f *Framework) cleanupAction() func() {
	resManager := f.ResourceManager
	return func() {
		if err := resManager.Cleanup(context.TODO()); err != nil {
			utils.Failf("%v", err)
		}
	}
}

func (f *Framework) buildRestConfig() (*rest.Config, error) {
	restCfg, err := clientcmd.BuildConfigFromFlags("", f.Options.KubeConfig)
	if err != nil {
		return nil, err
	}
	restCfg.QPS = 20
	restCfg.Burst = 50
	return restCfg, nil
}
