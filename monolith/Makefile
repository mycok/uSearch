# Note: Setup, build, push and deploy a usearch monolith application
# Steps:
# 	1. Apply k8s deployment namespaces.
#	2. Add and install helm charts for elasticsearch and cockroachdb data stores.
#	3. Build and push both the cockroachdb schema and usearch monolith app images.
#	4. Apply both cockroachdb schema and usearch monolith k8s manifests.

include ./monolith/k8s/Makefile
include ./monolith/cdb-schema/Makefile

.PHONY: containerize-app push-app containerize-and-push-app

SHELL = /bin/bash -o pipefail
IMAGE = usearch-monolith
SHA = $(shell git rev-parse --short HEAD)

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

containerize-and-push-app: containerize push

containerize-app:
	@echo "[docker build].....building ${IMAGE} (tags: ${PREFIX}${IMAGE}:latest, ${PREFIX}${IMAGE}:${SHA})....."
	@docker build --file ./monolith/Dockerfile \
		--tag ${PREFIX}${IMAGE}:latest \
		--tag ${PREFIX}${IMAGE}:${SHA} \
		. 2>&1 | sed -e "s/^/ | /g"

push-app:
	@echo "[docker push].....pushing ${PREFIX}${IMAGE}:latest....."
	@docker push ${PREFIX}${IMAGE}:latest 2>&1 | sed -e "s/^/ | /g"
	@echo "[docker push].....pushing ${PREFIX}${IMAGE}:{SHA}....."
	@docker push ${PREFIX}${IMAGE}:${SHA} 2>&1 | sed -e "s/^/ | /g"


