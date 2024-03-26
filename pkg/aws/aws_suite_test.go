// SPDX-FileCopyrightText: 2024 SAP SE or an SAP affiliate company and Gardener contributors
//
// SPDX-License-Identifier: Apache-2.0

package aws

import (
	"fmt"
	"testing"

	v1alpha1 "github.com/gardener/machine-controller-manager/pkg/apis/machine/v1alpha1"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
)

func TestAws(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Aws Suite")
}

const (
	testNamespace = "test"
)

// params is used to pass annotations to the machine spec
func newMachine(
	setMachineIndex int, annotations map[string]string,
) *v1alpha1.Machine {
	index := 0

	if setMachineIndex > 0 {
		index = setMachineIndex
	}

	machine := &v1alpha1.Machine{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "machine.sapcloud.io",
			Kind:       "Machine",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      fmt.Sprintf("machine-%d", index),
			Namespace: testNamespace,
		},
	}

	// Don't initialize providerID if setMachineIndex == -1
	if setMachineIndex != -1 {
		machine.Spec = v1alpha1.MachineSpec{
			ProviderID: fmt.Sprintf("aws:///eu-west-1/i-0123456789-%d", setMachineIndex),
		}
		machine.Labels = map[string]string{
			v1alpha1.NodeLabelKey: fmt.Sprintf("ip-%d", setMachineIndex),
		}
	}

	machine.Spec.NodeTemplateSpec.ObjectMeta.Annotations = make(map[string]string)

	//appending to already existing annotations
	for k, v := range annotations {
		machine.Spec.NodeTemplateSpec.ObjectMeta.Annotations[k] = v
	}
	return machine
}

func newMachineClass(providerSpec []byte) *v1alpha1.MachineClass {
	return &v1alpha1.MachineClass{
		ProviderSpec: runtime.RawExtension{
			Raw: providerSpec,
		},
		Provider: ProviderAWS,
	}
}

func newMachineClassWithProvider(providerSpec []byte, provider string) *v1alpha1.MachineClass {
	return &v1alpha1.MachineClass{
		ProviderSpec: runtime.RawExtension{
			Raw: providerSpec,
		},
		Provider: provider,
	}
}
