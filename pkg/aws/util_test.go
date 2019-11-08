/*
Copyright (c) 2019 SAP SE or an SAP affiliate company. All rights reserved.
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

package aws

import (
	api "github.com/gardener/machine-controller-manager-provider-aws/pkg/aws/apis"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/ginkgo/extensions/table"
	. "github.com/onsi/gomega"
)

var _ = Describe("Util", func() {

	Describe("#NewSession", func() {
		DescribeTable("##table",
			func(secret *api.Secrets) {

				region := "eu-west-1"
				plugin := pluginSPIImpl{}
				_, err := plugin.NewSession(*secret, region)
				Expect(err).ToNot(HaveOccurred())
			},
			Entry("Trying to create a new AWS Session with dummy values",
				&api.Secrets{
					ProviderAccessKeyID:     "dummy-id",
					ProviderSecretAccessKey: "dummy-secret",
					UserData:                "dummy-user-data",
				}),
			Entry("Trying to create a new AWS Session with default provider keys",
				&api.Secrets{
					UserData: "dummy-user-data",
				}),
		)
	})

	Describe("#NewEC2API", func() {
		DescribeTable("##table",
			func(secret *api.Secrets) {
				region := "eu-west-2"
				plugin := pluginSPIImpl{}
				session, err := plugin.NewSession(*secret, region)
				Expect(err).ToNot(HaveOccurred())
				EC2API := plugin.NewEC2API(session)
				Expect(EC2API).NotTo(BeNil())
			},
			Entry("Trying to create a new EC2API Interface with dummy values",
				&api.Secrets{
					ProviderAccessKeyID:     "dummy-id",
					ProviderSecretAccessKey: "dummy-secret",
					UserData:                "dummy-user-data",
				}),
		)
	})
})
