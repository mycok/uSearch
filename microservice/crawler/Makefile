.PHONY: containerize-crawler-service push-crawler-service containerize-and-push-crawler-service

SHELL = /bin/bash -o pipefail
CRAWLER_IMAGE = usearch-crawler
CRAWLER_SHA = $(shell git rev-parse --short HEAD)

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

containerize-and-push-crawler-service: containerize-crawler-service push-crawler-service

containerize-crawler-service:
	@echo "[docker build].....building ${CRAWLER_IMAGE} (tags: ${PREFIX}${CRAWLER_IMAGE}:latest, ${PREFIX}${CRAWLER_IMAGE}:${CRAWLER_SHA})....."
	@docker build --file ./microservice/crawler/Dockerfile \
		--tag ${PREFIX}${CRAWLER_IMAGE}:latest \
		--tag ${PREFIX}${CRAWLER_IMAGE}:${CRAWLER_SHA} \
		. 2>&1 | sed -e "s/^/ | /g"

push-crawler-service:
	@echo "[docker push].....pushing ${PREFIX}${CRAWLER_IMAGE}:latest....."
	@docker push ${PREFIX}${CRAWLER_IMAGE}:latest 2>&1 | sed -e "s/^/ | /g"
	@echo "[docker push].....pushing ${PREFIX}${CRAWLER_IMAGE}:${CRAWLER_SHA}....."
	@docker push ${PREFIX}${CRAWLER_IMAGE}:${CRAWLER_SHA} 2>&1 | sed -e "s/^/ | /g"


