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

PROVIDER_NAME       := AWS
PROJECT_NAME        := gardener
BINARY_PATH         := bin/
IMAGE_REPOSITORY    := docker-repository-link-goes-here
IMAGE_TAG           := $(shell cat VERSION)

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
# Rules for starting cmi-server locally
#########################################

.PHONY: start
start:
	go run app/aws/cmi-server.go --endpoint=tcp://127.0.0.1:8080

#########################################
# Rules for re-vendoring
#########################################

.PHONY: revendor
revendor:
	@dep ensure -v --update

#########################################
# Rules for testing
#########################################

.PHONY: test-unit
test-unit:
	.ci/test

#########################################
# Rules for build/release
#########################################

.PHONY: release
release: build-local build docker-image docker-push rename-binaries

.PHONY: build-local
build-local:
	go build \
	-v \
	-o ${BINARY_PATH}/cmi-server \
	app/controller/cmi-server.go

.PHONY: build
build:
	CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build \
	-a \
	-v \
	-o ${BINARY_PATH}/rel/cmi-server \
	app/controller/cmi-server.go

.PHONY: docker-image
docker-image:
	@if [[ ! -f ${BINARY_PATH}/rel/cmi-server ]]; then echo "No binary found. Please run 'make build'"; false; fi
	@docker build -t $(IMAGE_REPOSITORY):$(IMAGE_TAG) .

.PHONY: docker-push
docker-push:
	@if ! docker images $(IMAGE_REPOSITORY) | awk '{ print $$2 }' | grep -q -F $(IMAGE_TAG); then echo "$(IMAGE_REPOSITORY) version $(IMAGE_TAG) is not yet built. Please run 'make docker-images'"; false; fi
	@gcloud docker -- push $(IMAGE_REPOSITORY):$(IMAGE_TAG)

.PHONY: rename-binaries
rename-binaries:
	@if [[ -f bin/cmi-server ]]; then cp bin/cmi-server cmi-server-darwin-amd64; fi
	@if [[ -f bin/rel/cmi-server ]]; then cp bin/rel/cmi-server cmi-server-linux-amd64; fi

.PHONY: clean
clean:
	@rm -rf bin/
	@rm -f *linux-amd64
	@rm -f *darwin-amd64
