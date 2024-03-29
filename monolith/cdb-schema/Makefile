.PHONY: containerize-cdb push-cdb containerize-and-push-cdb

SHELL = /bin/bash -o pipefail
CDB_IMAGE = cdb-schema
CDB_SHA = $(shell git rev-parse --short HEAD)

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

containerize-and-push-cdb: containerize-cdb push-cdb

containerize-cdb:
	@echo "[docker build].....building ${CDB_IMAGE} (tags: ${PREFIX}${CDB_IMAGE}:latest, ${PREFIX}${CDB_IMAGE}:${CDB_SHA})....."
	@docker build --file ./monolith/cdb-schema/Dockerfile \
		--tag ${PREFIX}${CDB_IMAGE}:latest \
		--tag ${PREFIX}${CDB_IMAGE}:${CDB_SHA} \
		. 2>&1 | sed -e "s/^/ | /g"

push-cdb:
	@echo "[docker push].....pushing ${PREFIX}${CDB_IMAGE}:latest....."
	@docker push ${PREFIX}${CDB_IMAGE}:latest 2>&1 | sed -e "s/^/ | /g"
	@echo "[docker push].....pushing ${PREFIX}${CDB_IMAGE}:${CDB_SHA}....."
	@docker push ${PREFIX}${CDB_IMAGE}:${CDB_SHA} 2>&1 | sed -e "s/^/ | /g"
