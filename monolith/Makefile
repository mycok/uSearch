# Note: Setup, build, push and deploy a usearch monolith application
# Steps:
# 	1. Apply k8s deployment namespaces.
#	2. Add and install helm charts for elasticsearch and cockroachdb data stores.
#	3. Build and push both the cockroachdb schema and usearch monolith app images.
#	4. Apply both cockroachdb schema and usearch monolith k8s manifests.

include ./monolith/k8s/Makefile
include ./monolith/cdb-schema/Makefile

.PHONY: containerize-monolith-app push-monolith-app containerize-and-push-monolith-app

SHELL = /bin/bash -o pipefail
MONO_IMAGE = usearch-monolith
MONO_SHA = $(shell git rev-parse --short HEAD)

# Check if the value of the [PRIVATE_CONTAINER_REGISTRY_ADDR] variable is empty
# or undefined.
ifeq ($(origin PRIVATE_CONTAINER_REGISTRY_ADDR), undefined)
	PRIVATE_CONTAINER_REGISTRY_ADDR := $(shell minikube ip 2</dev/null):5000
endif

# Check if the value of the [PRIVATE_CONTAINER_REGISTRY_ADDR] variable is not empty.
ifneq ($(PRIVATE_CONTAINER_REGISTRY_ADDR),)
	# Use minikube as a private local image repository.
	PREFIX := ${PRIVATE_CONTAINER_REGISTRY_ADDR}/
else
	# Use docker-hub as a private local image repository.
	PREFIX := myckie/
endif

containerize-and-push-monolith-app: containerize-monolith-app push-monolith-app

containerize-monolith-app:
	@echo "[docker build].....building ${MONO_IMAGE} (tags: ${PREFIX}${MONO_IMAGE}:latest, ${PREFIX}${MONO_IMAGE}:${MONO_SHA})....."
	@docker build --file ./monolith/Dockerfile \
		--tag ${PREFIX}${MONO_IMAGE}:latest \
		--tag ${PREFIX}${MONO_IMAGE}:${MONO_SHA} \
		. 2>&1 | sed -e "s/^/ | /g"

push-monolith-app:
	@echo "[docker push].....pushing ${PREFIX}${MONO_IMAGE}:latest....."
	@docker push ${PREFIX}${MONO_IMAGE}:latest 2>&1 | sed -e "s/^/ | /g"
	@echo "[docker push].....pushing ${PREFIX}${MONO_IMAGE}:${MONO_SHA}....."
	@docker push ${PREFIX}${MONO_IMAGE}:${MONO_SHA} 2>&1 | sed -e "s/^/ | /g"


