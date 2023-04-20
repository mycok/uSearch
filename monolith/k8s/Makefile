.PHONY: uninstall deploy k8s-usearch-app-and-data-namespaces helm-es-service helm-cdb-service

SHELL=/bin/bash -o pipefail

ES_VERSION ?= 7.4.0
CDB_VERSION ?= v19.1.5

uninstall:
	@echo "[helm] uninstalling helm deployments for CDB/ES"
	@helm uninstall --purge es 2>&1 | sed -e 's/^/ | /g' || true
	@helm uninstall --purge cdb 2>&1 | sed -e 's/^/ | /g' || true

	@echo "[kubectl] deleting k8s deployments for CDB/ES"
	@kubectl delete -f /monolith/k8s/ 2>&1 | sed -e 's/^/ | /g' || true

deploy: k8s-usearch-app-and-data-namespaces helm-es-service helm-cdb-service
	@echo "[kubectl] deploying usearch app"
	@kubectl apply -f /monolith/k8s/ 2>&1 | sed -e 's/^/ | /g'

k8s-usearch-app-and-data-namespaces:
	@if [[ -z `kubectl get ns | grep data`]]; then \
		echo "[kubectl] applying namespaces manifest"; \
		kubectl apply -f /monolith/k8s/01-namespaces.yaml 2>&1 | sed -e 's/^/ | /g'; \
	fi

helm-es-service: k8s-usearch-app-and-data-namespaces
	@if [[ `kubectl -n usearch-data get pods -l release=es 2>/dev/null | wc -l` -eq '0']]; then \
		echo "[helm] adding elasticsearch chart to the repository"; \
		helm repo add elastic https://helm.elastic.co 2>&1 | sed -e 's/^/ | /g'; \
		echo "[helm] installing elasticsearch (${ES_VERSION})"; \
		helm install --namespace=usearch-data --name es \
			--values /monolith/k8s/chart-settings/es-settings.yaml \
			--set imageTag=${ES_VERSION} \
			elastic/elasticsearch 2>&1 | sed -e 's/^/ | /g'; \
	fi

helm-cdb-service: k8s-usearch-app-and-data-namespaces
	@if [[ `kubectl -n usearch-data get pods -l release=cdb 2>/dev/null | wc -l` -eq '0']]; then \
		echo "[helm] adding cockroachdb chart to the repository"; \
		helm repo add cockroachdb https://charts.cockroachdb.com/ 2>&1 | sed -e 's/^/ | /g'; \
		echo "[helm] installing cockroachdb (${CDB_VERSION})"; \
		helm install --namespace=usearch-data --name cbd \
			--values /monolith/k8s/chart-settings/cbd-settings.yaml \
			--set imageTag=${CDB_VERSION} \
			cockroachdb/cockroachdb 2>&1 | sed -e 's/^/ | /g' ;\
	fi

