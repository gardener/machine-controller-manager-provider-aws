package helpers

import (
	mcmClientset "github.com/gardener/machine-controller-manager/pkg/client/clientset/versioned"

	apiextensionsclientset "k8s.io/apiextensions-apiserver/pkg/client/clientset/clientset"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
)

//Cluster type to hold cluster specific details
type Cluster struct {
	restConfig          *rest.Config
	clientset           *kubernetes.Clientset
	apiextensionsClient *apiextensionsclientset.Clientset
	McmClient           *mcmClientset.Clientset
}

// FillClientSets checks whether the cluster is accessible and returns an error if not
func (c *Cluster) FillClientSets() error {
	clientset, err := kubernetes.NewForConfig(c.restConfig)
	if err == nil {
		c.clientset = clientset
		err = c.ProbeNodes()
		if err != nil {
			return err
		}
		apiextensionsClient, err := apiextensionsclientset.NewForConfig(c.restConfig)
		if err == nil {
			c.apiextensionsClient = apiextensionsClient
		}
		mcmClient, err := mcmClientset.NewForConfig(c.restConfig)
		if err == nil {
			c.McmClient = mcmClient
		}
	}
	return err
}

// NewCluster returns a Cluster struct
func NewCluster(kubeConfigPath string) (c *Cluster, e error) {
	config, err := clientcmd.BuildConfigFromFlags("", kubeConfigPath)
	if err == nil {
		c = &Cluster{
			restConfig: config,
		}
	} else {
		c = &Cluster{}
	}

	return c, err
}
