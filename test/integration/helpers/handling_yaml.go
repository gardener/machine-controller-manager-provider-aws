package helpers

import (
	"fmt"
	"io/ioutil"
	"log"
	"regexp"
	"strings"

	v1alpha1 "github.com/gardener/machine-controller-manager/pkg/apis/machine/v1alpha1"
	mcmscheme "github.com/gardener/machine-controller-manager/pkg/client/clientset/versioned/scheme"
	"k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1beta1"
	apiextensionsscheme "k8s.io/apiextensions-apiserver/pkg/client/clientset/clientset/scheme"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
)

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
				crd := obj.(*v1beta1.CustomResourceDefinition)
				_, err := c.apiextensionsClient.ApiextensionsV1beta1().CustomResourceDefinitions().Create(crd)
				if err != nil {
					return err
				}
			case "MachineClass":
				crd := obj.(*v1alpha1.MachineClass)
				_, err := c.mcmClient.MachineV1alpha1().MachineClasses("default").Create(crd)
				if err != nil {
					return err
				}
			case "Machine":
				crd := obj.(*v1alpha1.Machine)
				_, err := c.mcmClient.MachineV1alpha1().Machines("default").Create(crd)
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
