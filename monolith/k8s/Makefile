.PHONY: uninstall-monolith deploy-monolith k8s-monolith-usearch-app-and-data-namespaces helm-monolith-es-service helm-monolith-cdb-service

SHELL=/bin/bash -o pipefail

ES_VERSION ?= 8.5.1
CDB_VERSION ?= v19.1.5

uninstall-monolith:
	@echo "[helm] uninstalling helm deployments for CDB/ES"
	@helm uninstall --namespace=usearch-data es 2>&1 | sed -e 's/^/ | /g' || true
	@helm uninstall --namespace=usearch-data cdb 2>&1 | sed -e 's/^/ | /g' || true

	@echo "[kubectl] deleting k8s deployments for CDB/ES"
	@kubectl delete -f ./monolith/k8s/ 2>&1 | sed -e 's/^/ | /g' || true

deploy-monolith: k8s-usearch-app-and-data-namespaces helm-es-service helm-cdb-service
	@echo "[kubectl] deploying usearch app"
	@kubectl apply -f ./monolith/k8s/ 2>&1 | sed -e 's/^/ | /g'

k8s-monolith-usearch-app-and-data-namespaces:
	@if [[ -z `kubectl get ns | grep data` ]]; then \
		echo "[kubectl] applying namespaces manifest"; \
		kubectl apply -f ./monolith/k8s/01-namespaces.yaml 2>&1 | sed -e 's/^/ | /g'; \
	fi

helm-monolith-es-service: k8s-usearch-app-and-data-namespaces
	@if [[ `kubectl -n usearch-data get pods -l release=es 2>/dev/null | wc -l` -eq '0' ]]; then \
		echo "[helm] adding elasticsearch chart to the repository"; \
		helm repo add elastic https://helm.elastic.co 2>&1 | sed -e 's/^/ | /g'; \
		echo "[helm] installing elasticsearch (${ES_VERSION})"; \
		helm upgrade --install --namespace=usearch-data \
			--values ./monolith/k8s/chart-settings/es-settings.yaml \
			--set imageTag=${ES_VERSION} \
			es elastic/elasticsearch 2>&1 | sed -e 's/^/ | /g'; \
	fi

helm-monolith-cdb-service: k8s-usearch-app-and-data-namespaces
	@if [[ `kubectl -n usearch-data get pods -l release=cdb 2>/dev/null | wc -l` -eq '0' ]]; then \
		echo "[helm] adding cockroachdb chart to the repository"; \
		helm repo add cockroachdb https://charts.cockroachdb.com/ 2>&1 | sed -e 's/^/ | /g'; \
		echo "[helm] installing cockroachdb (${CDB_VERSION})"; \
		helm upgrade --install --namespace=usearch-data \
			--values ./monolith/k8s/chart-settings/cdb-settings.yaml \
			--set imageTag=${CDB_VERSION} \
			cdb cockroachdb/cockroachdb 2>&1 | sed -e 's/^/ | /g' ;\
	fi

