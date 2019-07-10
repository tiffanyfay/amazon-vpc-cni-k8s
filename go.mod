module github.com/aws/amazon-vpc-cni-k8s

go 1.12

require (
	github.com/Microsoft/go-winio v0.4.11 // indirect
<<<<<<< HEAD
	github.com/aws/aws-sdk-go v1.21.7
	github.com/beorn7/perks v0.0.0-20180321164747-3a771d992973 // indirect
=======
	github.com/aws/aws-sdk-go v1.19.21
>>>>>>> Fri31 works with 1 node and prow
	github.com/cihub/seelog v0.0.0-20151216151435-d2c6e5aa9fbf
	github.com/containernetworking/cni v0.5.2
	github.com/coreos/go-iptables v0.4.0
	github.com/deckarep/golang-set v1.7.1
	github.com/docker/distribution v2.6.2+incompatible // indirect
	github.com/docker/docker v1.13.1
	github.com/docker/go-connections v0.4.0 // indirect
	github.com/docker/go-units v0.3.3 // indirect
	github.com/ghodss/yaml v1.0.0 // indirect
	github.com/golang/glog v0.0.0-20160126235308-23def4e6c14b
	github.com/golang/mock v1.2.0
	github.com/golang/protobuf v1.3.1
	github.com/google/btree v1.0.0 // indirect
	github.com/kubernetes-sigs/aws-alb-ingress-controller v1.1.2
	github.com/onsi/ginkgo v1.8.0
	github.com/onsi/gomega v1.5.0
	github.com/operator-framework/operator-sdk v0.0.7
	github.com/pkg/errors v0.8.1
	github.com/prometheus/client_golang v0.9.3
	github.com/prometheus/client_model v0.0.0-20190129233127-fd36f4220a90
	github.com/prometheus/common v0.4.0
	github.com/spf13/pflag v1.0.3
	github.com/stevvooe/resumable v0.0.0-20180830230917-22b14a53ba50 // indirect
	github.com/stretchr/testify v1.2.2
	github.com/vishvananda/netlink v1.0.0
	github.com/vishvananda/netns v0.0.0-20180720170159-13995c7128cc // indirect
	golang.org/x/net v0.0.0-20190108225652-1e06a53dbb7e
	golang.org/x/sync v0.0.0-20190423024810-112230192c58 // indirect
	golang.org/x/sys v0.0.0-20190214214411-e77772198cdc
	google.golang.org/genproto v0.0.0-20180817151627-c66870c02cf8 // indirect
	google.golang.org/grpc v1.14.0
	k8s.io/api v0.0.0-20181213150558-05914d821849
	k8s.io/apimachinery v0.0.0-20181127025237-2b1284ed4c93
	k8s.io/client-go v2.0.0-alpha.0.0.20181213151034-8d9ed539ba31+incompatible
	k8s.io/kube-openapi v0.0.0-20190510232812-a01b7d5d6c22 // indirect
)
