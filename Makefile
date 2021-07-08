# Copyright (c) 2019 SAP SE or an SAP affiliate company. All rights reserved.
#
# Licensed under the Apache License, Version 2.0 (the "License");
# you may not use this file except in compliance with the License.
# You may obtain a copy of the License at
#
#      http://www.apache.org/licenses/LICENSE-2.0
#
# Unless required by applicable law or agreed to in writing, software
# distributed under the License is distributed on an "AS IS" BASIS,
# WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
# See the License for the specific language governing permissions and
# limitations under the License.

BINARY_PATH         := bin/
COVERPROFILE        := test/output/coverprofile.out
IMAGE_REPOSITORY    := eu.gcr.io/gardener-project/gardener/machine-controller-manager-provider-aws
IMAGE_TAG           := $(shell cat VERSION)
PROVIDER_NAME       := AWS
PROJECT_NAME        := gardener
CONTROL_KUBECONFIG  := dev/control_kubeconfig.yaml
TARGET_KUBECONFIG   := dev/target_kubeconfig.yaml

# Below ones are used in tests
MACHINECLASS_V1 	:= dev/machineclassv1.yaml
MACHINECLASS_V2 	:= 
MCM_IMAGE			:=
MC_IMAGE			:=
# MCM_IMAGE			:= eu.gcr.io/gardener-project/gardener/machine-controller-manager:v0.39.0
# MC_IMAGE			:= $(IMAGE_REPOSITORY):v0.7.0
LEADER_ELECT 	    := "true"

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
			-mod=vendor \
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
# Rules for re-vendoring
#########################################

.PHONY: revendor
revendor:
	@env GO111MODULE=on go mod vendor -v
	@env GO111MODULE=on go mod tidy -v

.PHONY: update-dependencies
update-dependencies:
	@env GO111MODULE=on go get -u

#########################################
# Rules for testing
#########################################

.PHONY: test-unit
test-unit:
	.ci/test

.PHONY: test-integration
test-integration:
	@if [[ -f $(PWD)/$(CONTROL_KUBECONFIG) ]]; then export CONTROL_KUBECONFIG=$(PWD)/$(CONTROL_KUBECONFIG); fi; \
	if [[ -f $(PWD)/$(TARGET_KUBECONFIG) ]]; then export TARGET_KUBECONFIG=$(PWD)/$(TARGET_KUBECONFIG); fi; \
	if [[ -f $(PWD)/$(MACHINECLASS_V1) ]]; then export MACHINECLASS_V1=$(PWD)/$(MACHINECLASS_V1); fi; \
	if [[ -f $(PWD)/$(MACHINECLASS_V2) ]]; then export MACHINECLASS_V2=$(PWD)/$(MACHINECLASS_V2); fi; \
	export MC_CONTAINER_IMAGE=$(MC_IMAGE); \
	export MCM_CONTAINER_IMAGE=$(MCM_IMAGE); \
	export CONTROL_CLUSTER_NAMESPACE=$(CONTROL_NAMESPACE); \
	export MACHINE_CONTROLLER_MANAGER_DEPLOYMENT_NAME=$(MACHINE_CONTROLLER_MANAGER_DEPLOYMENT_NAME); \
	.ci/integration_test

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

.PHONY: docker-image
docker-image:
	@docker build -t $(IMAGE_REPOSITORY):$(IMAGE_TAG) .

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
