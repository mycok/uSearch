.PHONY: containerize-frontend-service push-frontend-service containerize-and-push-frontend-service

SHELL = /bin/bash -o pipefail
FRONTEND_IMAGE = usearch-frontend
FRONTEND_SHA = $(shell git rev-parse --short HEAD)

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

containerize-and-push-frontend-service: containerize-frontend-service push-frontend-service

containerize-frontend-service:
	@echo "[docker build].....building ${FRONTEND_IMAGE} (tags: ${PREFIX}${FRONTEND_IMAGE}:latest, ${PREFIX}${FRONTEND_IMAGE}:${FRONTEND_SHA})....."
	@docker build --file ./microservice/frontend/Dockerfile \
		--tag ${PREFIX}${FRONTEND_IMAGE}:latest \
		--tag ${PREFIX}${FRONTEND_IMAGE}:${FRONTEND_SHA} \
		. 2>&1 | sed -e "s/^/ | /g"

push-frontend-service:
	@echo "[docker push].....pushing ${PREFIX}${FRONTEND_IMAGE}:latest....."
	@docker push ${PREFIX}${FRONTEND_IMAGE}:latest 2>&1 | sed -e "s/^/ | /g"
	@echo "[docker push].....pushing ${PREFIX}${FRONTEND_IMAGE}:${FRONTEND_SHA}....."
	@docker push ${PREFIX}${FRONTEND_IMAGE}:${FRONTEND_SHA} 2>&1 | sed -e "s/^/ | /g"


