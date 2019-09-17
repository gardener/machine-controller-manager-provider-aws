module github.com/gardener/machine-controller-manager-provider-aws

go 1.12

require (
	github.com/aws/aws-sdk-go v1.20.12
	github.com/container-storage-interface/spec v0.3.0
	github.com/gardener/machine-spec v0.0.0-20190916105733-70a34003d1c6
	github.com/golang/glog v0.0.0-20160126235308-23def4e6c14b
	github.com/onsi/ginkgo v1.8.0
	github.com/onsi/gomega v1.5.0
	github.com/spf13/cobra v0.0.5
	golang.org/x/net v0.0.0-20190628185345-da137c7871d7
	golang.org/x/sys v0.0.0-20190626221950-04f50cda93cb // indirect
	google.golang.org/genproto v0.0.0-20190627203621-eb59cef1c072 // indirect
	google.golang.org/grpc v1.21.1
	k8s.io/api v0.0.0-20190718062839-c8a0b81cb10e
)
