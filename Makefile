# SPDX-FileCopyrightText: 2024 SAP SE or an SAP affiliate company and Gardener contributors
#
# SPDX-License-Identifier: Apache-2.0
ENSURE_MCM_MOD := $(shell go mod download github.com/gardener/machine-controller-manager)
MCM_DIR   	:= $(shell go list -m -f "{{.Dir}}" github.com/gardener/machine-controller-manager)
TOOLS_DIR := hack/tools
include $(MCM_DIR)/hack/tools.mk
-include .env
export
BINARY_PATH         := bin/
COVERPROFILE        := test/output/coverprofile.out
IMAGE_REPOSITORY    := europe-docker.pkg.dev/gardener-project/public/gardener/machine-controller-manager-provider-aws
IMAGE_TAG           := $(shell cat VERSION)
PROVIDER_NAME       := AWS
PROJECT_NAME        := gardener
LEADER_ELECT 	    := "true"
# If Integration Test Suite is to be run locally against clusters then export the below variable
# with MCM deployment name in the cluster
MACHINE_CONTROLLER_MANAGER_DEPLOYMENT_NAME := machine-controller-manager
#########################################
# Rules for running helper scripts
#########################################

.PHONY: rename-provider
rename-provider:
	@./hack/rename-provider ${PROVIDER_NAME}

.PHONY: rename-project
rename-project:
	@./hack/rename-project ${PROJECT_NAME}

#########################################
# Rules for starting machine-controller locally
#########################################

.PHONY: start
start:
	@GO111MODULE=on go run \
			cmd/machine-controller/main.go \
			--control-kubeconfig=$(CONTROL_KUBECONFIG) \
			--target-kubeconfig=$(TARGET_KUBECONFIG) \
			--namespace=$(CONTROL_NAMESPACE) \
			--machine-creation-timeout=20m \
			--machine-drain-timeout=5m \
			--machine-health-timeout=10m \
			--machine-pv-detach-timeout=2m \
			--machine-safety-apiserver-statuscheck-timeout=30s \
			--machine-safety-apiserver-statuscheck-period=1m \
			--machine-safety-orphan-vms-period=30m \
			--leader-elect=$(LEADER_ELECT) \
			--v=3

#########################################
# Rules for checks
#########################################

.PHONY: check
check:
	.ci/check

#########################################
# Rules for tidying
#########################################

.PHONY: tidy
tidy:
	@env GO111MODULE=on go mod tidy -v

.PHONY: update-dependencies
update-dependencies:
	@env GO111MODULE=on go get -u

#########################################
# Rules for testing
#########################################

.PHONY: test-unit
test-unit:
	@SKIP_INTEGRATION_TESTS=X .ci/test

.PHONY: test-integration
test-integration:
	.ci/local_integration_test

.PHONY: test
test:
	.ci/test

#########################################
# Rules for build/release
#########################################

.PHONY: release
release: build-local build docker-image docker-login docker-push rename-binaries

.PHONY: build-local
build-local:
	@env LOCAL_BUILD=1 .ci/build

.PHONY: build
build:
	@.ci/build

PLATFORM ?= linux/amd64
.PHONY: docker-image
docker-image:
	@docker buildx build --platform $(PLATFORM) -t $(IMAGE_REPOSITORY):$(IMAGE_TAG) .

.PHONY: docker-login
docker-login:
	@gcloud auth login

.PHONY: docker-push
docker-push:
	@if ! docker images $(IMAGE_REPOSITORY) | awk '{ print $$2 }' | grep -q -F $(IMAGE_TAG); then echo "$(IMAGE_REPOSITORY) version $(IMAGE_TAG) is not yet built. Please run 'make docker-images'"; false; fi
	@gcloud docker -- push $(IMAGE_REPOSITORY):$(IMAGE_TAG)

.PHONY: rename-binaries
rename-binaries:
	@if [[ -f bin/machine-controller ]]; then cp bin/machine-controller machine-controller-darwin-amd64; fi
	@if [[ -f bin/rel/machine-controller ]]; then cp bin/rel/machine-controller machine-controller-linux-amd64; fi

.PHONY: clean
clean:
	@rm -rf bin/
	@rm -f *linux-amd64
	@rm -f *darwin-amd64

.PHONY: add-license-headers
add-license-headers: $(GO_ADD_LICENSE)
	@./hack/add_license_headers.sh ${YEAR}

.PHONY: sast
sast: $(GOSEC)
	@./hack/sast.sh

.PHONY: sast-report
sast-report: $(GOSEC)
	@./hack/sast.sh --gosec-report true

.PHONY: verify
verify: check build test
