.PHONY: containerize-pagerank-service push-pagerank-service containerize-and-push-pagerank-service

SHELL = /bin/bash -o pipefail
PAGERANK_IMAGE = usearch-pagerank
PAGERANK_SHA = $(shell git rev-parse --short HEAD)

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

containerize-and-push-pagerank-service: containerize-pagerank-service push-pagerank-service

containerize-pagerank-service:
	@echo "[docker build].....building ${PAGERANK_IMAGE} (tags: ${PREFIX}${PAGERANK_IMAGE}:latest, ${PREFIX}${PAGERANK_IMAGE}:${PAGERANK_SHA})....."
	@docker build --file ./microservice/pagerank/Dockerfile \
		--tag ${PREFIX}${PAGERANK_IMAGE}:latest \
		--tag ${PREFIX}${PAGERANK_IMAGE}:${PAGERANK_SHA} \
		. 2>&1 | sed -e "s/^/ | /g"

push-pagerank-service:
	@echo "[docker push].....pushing ${PREFIX}${PAGERANK_IMAGE}:latest....."
	@docker push ${PREFIX}${PAGERANK_IMAGE}:latest 2>&1 | sed -e "s/^/ | /g"
	@echo "[docker push].....pushing ${PREFIX}${PAGERANK_IMAGE}:${PAGERANK_SHA}....."
	@docker push ${PREFIX}${PAGERANK_IMAGE}:${PAGERANK_SHA} 2>&1 | sed -e "s/^/ | /g"


