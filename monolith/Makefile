.PHONY: containerize push containerize-and-push

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

containerize-and-push: containerize push

containerize:
	@echo "[docker build].....building ${IMAGE} (tags: ${PREFIX}${IMAGE}:latest, ${PREFIX}${IMAGE}:${SHA})....."
	@docker build --file ./monolith/Dockerfile \
		--tag ${PREFIX}${IMAGE}:latest \
		--tag ${PREFIX}${IMAGE}:${SHA} \
		. 2>&1 | sed -e "s/^/ | /g"

push:
	@echo "[docker push].....pushing ${PREFIX}${IMAGE}:latest....."
	@docker push ${PREFIX}${IMAGE}:latest 2>&1 | sed -e "s/^/ | /g"
	@echo "[docker push].....pushing ${PREFIX}${IMAGE}:{SHA}....."
	@docker push ${PREFIX}${IMAGE}:${SHA} 2>&1 | sed -e "s/^/ | /g"
