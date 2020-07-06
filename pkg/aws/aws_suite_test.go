/*
Copyright (c) 2020 SAP SE or an SAP affiliate company. All rights reserved.
Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at
    http://www.apache.org/licenses/LICENSE-2.0
Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/
package aws_test

import (
	"fmt"
	"testing"

	v1alpha1 "github.com/gardener/machine-controller-manager/pkg/apis/machine/v1alpha1"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestAws(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Aws Suite")
}

const (
	testNamespace = "test"
)

func newMachine() *v1alpha1.Machine {
	return newMachines(1)[0]
}

func newMachines(
	machineCount int,

) []*v1alpha1.Machine {
	machines := make([]*v1alpha1.Machine, machineCount)

	for i := range machines {
		m := &v1alpha1.Machine{
			TypeMeta: metav1.TypeMeta{
				APIVersion: "machine.sapcloud.io",
				Kind:       "Machine",
			},
			ObjectMeta: metav1.ObjectMeta{
				Name:      fmt.Sprintf("machine-%d", i),
				Namespace: testNamespace,
			},
		}

		machines[i] = m
	}
	return machines
}
