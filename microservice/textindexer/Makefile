.PHONY: containerize-textindexer-service push-textindexer-service containerize-and-push-textindexer-service

SHELL = /bin/bash -o pipefail
INDEXER_IMAGE = usearch-textindexer
INDEXER_SHA = $(shell git rev-parse --short HEAD)

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

containerize-and-push-textindexer-service: containerize-textindexer-service push-textindexer-service

containerize-textindexer-service:
	@echo "[docker build].....building ${INDEXER_IMAGE} (tags: ${PREFIX}${INDEXER_IMAGE}:latest, ${PREFIX}${INDEXER_IMAGE}:${INDEXER_SHA})....."
	@docker build --file ./microservice/textindexer/Dockerfile \
		--tag ${PREFIX}${INDEXER_IMAGE}:latest \
		--tag ${PREFIX}${INDEXER_IMAGE}:${INDEXER_SHA} \
		. 2>&1 | sed -e "s/^/ | /g"

push-textindexer-service:
	@echo "[docker push].....pushing ${PREFIX}${INDEXER_IMAGE}:latest....."
	@docker push ${PREFIX}${INDEXER_IMAGE}:latest 2>&1 | sed -e "s/^/ | /g"
	@echo "[docker push].....pushing ${PREFIX}${INDEXER_IMAGE}:${INDEXER_SHA}....."
	@docker push ${PREFIX}${INDEXER_IMAGE}:${INDEXER_SHA} 2>&1 | sed -e "s/^/ | /g"
