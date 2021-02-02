package helpers

import (
	"fmt"
	"io/ioutil"
	"log"
	"regexp"
	"strings"
	"time"

	v1alpha1 "github.com/gardener/machine-controller-manager/pkg/apis/machine/v1alpha1"
	mcmscheme "github.com/gardener/machine-controller-manager/pkg/client/clientset/versioned/scheme"
	"k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1beta1"
	apiextensionsv1beta1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1beta1"
	apiextensionsscheme "k8s.io/apiextensions-apiserver/pkg/client/clientset/clientset/scheme"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/util/wait"
)

var (
	crdCreated []string
)

// parsek8sYaml reads a yaml file and parses it based on the scheme
func parseK8sYaml(filepath string) ([]runtime.Object, []*schema.GroupVersionKind, error) {
	fileR, err := ioutil.ReadFile(filepath)
	if err != nil {
		return nil, nil, err
	}

	acceptedK8sTypes := regexp.MustCompile(`(Role|ClusterRole|RoleBinding|ClusterRoleBinding|ServiceAccount|CustomResourceDefinition)`)
	acceptedMCMTypes := regexp.MustCompile(`(MachineClass|Machine)`)
	fileAsString := string(fileR[:])
	sepYamlfiles := strings.Split(fileAsString, "---")
	retObj := make([]runtime.Object, 0, len(sepYamlfiles))
	retKind := make([]*schema.GroupVersionKind, 0, len(sepYamlfiles))
	for _, f := range sepYamlfiles {
		if f == "\n" || f == "" {
			// ignore empty cases
			continue
		}

		isExist, err := regexp.Match("CustomResourceDefinition", []byte(f))
		if isExist == true {
			decode := apiextensionsscheme.Codecs.UniversalDeserializer().Decode
			obj, groupVersionKind, err := decode([]byte(f), nil, nil)
			if err != nil {
				log.Println(fmt.Sprintf("Error while decoding YAML object. Err was: %s", err))
				continue
			}
			if !acceptedK8sTypes.MatchString(groupVersionKind.Kind) {
				log.Printf("The custom-roles configMap contained K8s object types which are not supported! Skipping object with type: %s", groupVersionKind.Kind)
			} else {
				retKind = append(retKind, groupVersionKind)
				retObj = append(retObj, obj)
			}
		} else if isExist == false {
			decode := mcmscheme.Codecs.UniversalDeserializer().Decode
			obj, groupVersionKind, err := decode([]byte(f), nil, nil)
			if err != nil {
				log.Println(fmt.Sprintf("Error while decoding YAML object. Err was: %s", err))
				continue
			}
			if !acceptedMCMTypes.MatchString(groupVersionKind.Kind) {
				log.Printf("The custom-roles configMap contained K8s object types which are not supported! Skipping object with type: %s", groupVersionKind.Kind)
			} else {
				retKind = append(retKind, groupVersionKind)
				retObj = append(retObj, obj)
			}
		} else {
			panic(err)
		}

	}
	return retObj, retKind, err
}

// ApplyYamlFile uses yaml to create resources in kubernetes
func (c *Cluster) ApplyYamlFile(filePath string) error {
	/* TO-DO: This function checks for the availability of filePath
	if available, then uses kubectl to perform kubectl apply on that file
	*/
	runtimeobj, kind, err := parseK8sYaml(filePath)
	if err == nil {
		for key, obj := range runtimeobj {
			switch kind[key].Kind {
			case "CustomResourceDefinition":
				crdName := obj.(*v1beta1.CustomResourceDefinition).Name
				crd := obj.(*v1beta1.CustomResourceDefinition)
				_, err := c.apiextensionsClient.ApiextensionsV1beta1().CustomResourceDefinitions().Create(crd)
				crdCreated = append(crdCreated, crdName)
				if err != nil {
					return err
				}
			case "MachineClass":
				crd := obj.(*v1alpha1.MachineClass)
				_, err := c.McmClient.MachineV1alpha1().MachineClasses("default").Create(crd)
				if err != nil {
					return err
				}
			case "Machine":
				crd := obj.(*v1alpha1.Machine)
				_, err := c.McmClient.MachineV1alpha1().Machines("default").Create(crd)
				if err != nil {
					return err
				}
			case "MachineDeployment":
				crd := obj.(*v1alpha1.MachineDeployment)
				_, err := c.McmClient.MachineV1alpha1().MachineDeployments("default").Create(crd)
				if err != nil {
					return err
				}
			}
		}
	} else {
		return err
	}

	return nil
}

// CheckEstablished uses the specified name to check if it is established
func (c *Cluster) CheckEstablished() error {
	/* TO-DO: This function checks the CRD for an established status
	if established status is not met, it will sleep until it meets
	*/

	for i, crdCreatedName := range crdCreated {
		fmt.Printf("%d Created CRD with given name %s, waiting till its established\n", i, crdCreatedName)

		err := wait.Poll(500*time.Millisecond, 60*time.Second, func() (bool, error) {
			crd, err := c.apiextensionsClient.ApiextensionsV1beta1().CustomResourceDefinitions().Get(crdCreatedName, metav1.GetOptions{})
			if err != nil {
				return false, err
			}
			for _, cond := range crd.Status.Conditions {
				switch cond.Type {
				case apiextensionsv1beta1.Established:
					if cond.Status == apiextensionsv1beta1.ConditionTrue {
						fmt.Printf("%d - %s, is established\n", i, crdCreatedName)
						return true, err
					}
				case apiextensionsv1beta1.NamesAccepted:
					if cond.Status == apiextensionsv1beta1.ConditionFalse {
						fmt.Printf("Name conflict: %v\n", cond.Reason)
						fmt.Printf("Naming Conflict with created CRD %s\n", crdCreatedName)
					}
				}
			}
			return false, err
		})
		if err != nil {
			return err
		}
	}

	return nil
}
