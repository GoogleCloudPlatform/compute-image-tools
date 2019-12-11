# Copyright 2016 The Kubernetes Authors.
#
# Licensed under the Apache License, Version 2.0 (the "License");
# you may not use this file except in compliance with the License.
# You may obtain a copy of the License at
#
#     http://www.apache.org/licenses/LICENSE-2.0
#
# Unless required by applicable law or agreed to in writing, software
# distributed under the License is distributed on an "AS IS" BASIS,
# WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
# See the License for the specific language governing permissions and
# limitations under the License.

# Variable defaults.
PROJECT       ?= compute-image-tools-test
BUILD_PROJECT ?= compute-image-tools-test
ZONE          ?= us-west1-b
CLUSTER       ?= prow

# Build and push specific variables.
REGISTRY ?= gcr.io
PUSH     ?= gcloud docker -- push

DOCKER_LABELS=--label io.k8s.prow.git-describe="$(shell git describe --tags --always --dirty)"

update-config: get-cluster-credentials
	kubectl create configmap config --from-file=config.yaml=config.yaml --dry-run -o yaml | kubectl replace configmap config -f -

update-plugins: get-cluster-credentials
	kubectl create configmap plugins --from-file=plugins.yaml=plugins.yaml --dry-run -o yaml | kubectl replace configmap plugins -f -

get-cluster-credentials:
	gcloud container clusters get-credentials "$(CLUSTER)" --project="$(PROJECT)" --zone="$(ZONE)"

get-build-cluster-credentials:
	gcloud container clusters get-credentials "$(CLUSTER)" --project="$(BUILD_PROJECT)" --zone="$(ZONE)"

.PHONY: update-config update-plugins get-cluster-credentials

hook-deployment: get-cluster-credentials
	kubectl apply -f cluster/hook_deployment.yaml

hook-service: get-cluster-credentials
	kubectl apply -f cluster/hook_service.yaml

sinker-deployment: get-cluster-credentials
	kubectl apply -f cluster/sinker_deployment.yaml

deck-deployment: get-cluster-credentials
	kubectl apply -f cluster/deck_deployment.yaml

deck-service: get-cluster-credentials
	kubectl apply -f cluster/deck_service.yaml

horologium-deployment: get-cluster-credentials
	kubectl apply -f cluster/horologium_deployment.yaml

plank-deployment: get-cluster-credentials
	kubectl apply -f cluster/plank_deployment.yaml

tide-deployment: get-cluster-credentials
	kubectl apply -f cluster/tide_deployment.yaml

tide-service: get-cluster-credentials
	kubectl apply -f cluster/tide_service.yaml

.PHONY: hook-deployment hook-service sinker-deployment deck-deployment deck-service horologium-deployment plank-deployment tide-deployment tide-service

update-all: update-config update-plugins hook-deployment hook-service sinker-deployment deck-deployment deck-service horologium-deployment plank-deployment tide-deployment tide-service
