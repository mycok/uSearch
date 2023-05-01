.PHONY: containerize-linkgraph-service push-linkgraph-service containerize-and-push-linkgraph-service

SHELL = /bin/bash -o pipefail
LINK_GRAPH_IMAGE = usearch-linkgraph
LINK_GRAPH_SHA = $(shell git rev-parse --short HEAD)

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

containerize-and-push-linkgraph-service: containerize-linkgraph-service push-linkgraph-service

containerize-linkgraph-service:
	@echo "[docker build].....building ${LINK_GRAPH_IMAGE} (tags: ${PREFIX}${LINK_GRAPH_IMAGE}:latest, ${PREFIX}${LINK_GRAPH_IMAGE}:${LINK_GRAPH_SHA})....."
	@docker build --file ./microservice/linkgraph/Dockerfile \
		--tag ${PREFIX}${LINK_GRAPH_IMAGE}:latest \
		--tag ${PREFIX}${LINK_GRAPH_IMAGE}:${LINK_GRAPH_SHA} \
		. 2>&1 | sed -e "s/^/ | /g"

push-linkgraph-service:
	@echo "[docker push].....pushing ${PREFIX}${LINK_GRAPH_IMAGE}:latest....."
	@docker push ${PREFIX}${LINK_GRAPH_IMAGE}:latest 2>&1 | sed -e "s/^/ | /g"
	@echo "[docker push].....pushing ${PREFIX}${LINK_GRAPH_IMAGE}:${LINK_GRAPH_SHA}....."
	@docker push ${PREFIX}${LINK_GRAPH_IMAGE}:${LINK_GRAPH_SHA} 2>&1 | sed -e "s/^/ | /g"
