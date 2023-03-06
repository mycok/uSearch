.PHONY: dockerize push dockerize-and-push

SHELL = /bin/bash -o pipefail
IMAGE = cdb-schema
SHA = $(shell git rev-parse --short HEAD)

# Check if the value of the [PRIVATE_CONTAINER_REGISTRY_ADDR] variable is empty
# or undefined.
ifeq ($(origin PRIVATE_CONTAINER_REGISTRY_ADDR), undefined)
	PRIVATE_CONTAINER_REGISTRY_ADDR := $(shell minikube api):5000
endif

# Check if the value of the [PRIVATE_CONTAINER_REGISTRY_ADDR] variable is empty.
ifneq ($(PRIVATE_CONTAINER_REGISTRY_ADDR),)
	PREFIX := ${PRIVATE_CONTAINER_REGISTRY_ADDR}/
endif

dockerize-and-push: dockerize push

dockerize:
	@echo "[docker build].....building ${IMAGE} (tags: ${PREFIX}${IMAGE}:latest, ${PREFIX}${IMAGE}:${SHA})....."
	@docker build --file ./Dockerfile \
		--tag ${PREFIX}${IMAGE}:latest \
		--tag ${PREFIX}${IMAGE}:${SHA} \
		../../ 2>&1 | sed -e "s/^/ | /g"

push:
	@echo "[docker push].....pushing ${PREFIX}${IMAGE}:latest....."
	@docker push ${PREFIX}${IMAGE}:latest 2>&1 | sed -e "s/^/ | /g"
	@echo "[docker push].....pushing ${PREFIX}${IMAGE}:{SHA}....."
	@docker push ${PREFIX}${IMAGE}:${SHA} 2>&1 | sed -e "s/^/ | /g"
