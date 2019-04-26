package aws

import (
	api "github.com/gardener/machine-controller-manager-provider-aws/pkg/aws/apis"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/ginkgo/extensions/table"
	. "github.com/onsi/gomega"
)

var _ = Describe("Util", func() {

	Describe("#NewSession", func() {
		type setup struct {
			secret api.Secrets
		}
		type action struct {
		}
		type expect struct {
		}
		type data struct {
			setup  setup
			action action
			expect expect
		}
		DescribeTable("##table",
			func(data *data) {

				region := "eu-west-1"
				driver := driverSPIImpl{}
				_, err := driver.NewSession(data.setup.secret, region)
				Expect(err).ToNot(HaveOccurred())
			},
			Entry("Trying to create a new AWS Session with dummy values", &data{
				setup: setup{
					secret: api.Secrets{
						ProviderAccessKeyID:     "dummy-id",
						ProviderSecretAccessKey: "dummy-secret",
						UserData:                "dummy-user-data",
					},
				},
			}),
			Entry("Trying to create a new AWS Session with default provider keys", &data{
				setup: setup{
					secret: api.Secrets{
						UserData: "dummy-user-data",
					},
				},
			}),
		)
	})

	Describe("#NewEC2API", func() {
		type setup struct {
			secret api.Secrets
		}
		type action struct {
		}
		type expect struct {
		}
		type data struct {
			setup  setup
			action action
			expect expect
		}
		DescribeTable("##table",
			func(data *data) {

				region := "eu-west-1"
				driver := driverSPIImpl{}
				session, err := driver.NewSession(data.setup.secret, region)
				Expect(err).ToNot(HaveOccurred())
				EC2API := driver.NewEC2API(session)
				Expect(EC2API).NotTo(BeNil())
			},
			Entry("Trying to create a new EC2API Interface with dummy values", &data{
				setup: setup{
					secret: api.Secrets{
						ProviderAccessKeyID:     "dummy-id",
						ProviderSecretAccessKey: "dummy-secret",
						UserData:                "dummy-user-data",
					},
				},
			}),
		)
	})
})
