module github.com/gardener/machine-controller-manager-provider-aws

go 1.16

require (
	github.com/aws/aws-sdk-go v1.36.0
	github.com/cenkalti/backoff/v4 v4.1.2
	github.com/gardener/machine-controller-manager v0.40.1-0.20210913052707-0051cd1cccd0
	github.com/onsi/ginkgo v1.16.2
	github.com/onsi/gomega v1.11.0
	github.com/spf13/pflag v1.0.5
	k8s.io/api v0.20.6
	k8s.io/apimachinery v0.20.6
	k8s.io/component-base v0.20.6
	k8s.io/klog v0.4.0
	k8s.io/utils v0.0.0-20201110183641-67b214c5f920
)
