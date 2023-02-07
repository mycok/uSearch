.PHONY: help deps test lint lint-check-deps

##	help: Print this help message
help:
		@echo 'Usage:'
		@sed -n 's/^##//p' ${MAKEFILE_LIST} | column -t -s ':' | sed -e 's/^/ /'

##	deps: Check for mod file and if missing, download the golang dep package
deps:
		@if [ "$(go mod help | echo 'no-mod')" = "no-mod" ]; then \
				echo "[dep] fetching package dependencies"; \
				go get -u github.com/golang/dep/cmd/dep; \
				dep ensure; \
		fi

##	test: Run all test suites with coverage metrics
test:
		@echo "[go test] running tests and collecting coverage metrics"
		@go test -v -tags all_tests -race -coverprofile=coverage.txt -covermode=atomic ./...

##	lint: Perform linting operations on all .go files in the root directory
lint: lint-check-deps
		@echo "[golangci-lint] linting sources"
		@golangci-lint run \
				--exclude-use-default=false \
				-v \
				-E gofmt \
				-E golint \
				-E govet \
				-E misspell \
				-E unconvert \
				./...

##	lint-check-deps: Check for existance of the linting packages such as [golangci-lint]
lint-check-deps:
		@if [ -z `which golangci-lint` ]; then \
				@echo "[go get] installing golangci-lint";\
				go get -u github.com/golangci/golangci-lint/cmd/golangci-lint; \
		fi

.PHONY: create-db-table run-db-up-migrations \
run-db-down-migations fix-dirty-db migrate-check-deps check-db-env

##	new-cdb-migrations: Create new migration files with the specified name passed as a commandline env var [name]
create-db-table: migrate-check-deps
		@echo '.....Creating migration files for ${name}.....'
		migrate create -seq -ext .sql -dir=./linkgraph/store/cdb/migrations ${name}

##	run-cdb-migations: Run all [up] migrations
run-db-up-migations: migrate-check-deps
		migrate -path ./linkgraph/store/cdb/migrations -database '$(subst postgresql,cockroachdb,${CDB_DSN})' up ${version}

##	run-cdb-migations: Run all [down] migrations
run-db-down-migations: migrate-check-deps
		migrate -path ./linkgraph/store/cdb/migrations -database '$(subst postgresql,cockroachdb,${CDB_DSN})' down ${version}

##	fix-dirty-db: Cleans and restores the database to a usable / clean state.
fix-dirty-db: migrate-check-deps
		migrate -path ./linkgraph/store/cdb/migrations -database '$(subst postgresql,cockroachdb,${CDB_DSN})' force ${version}

##	migrate-check-deps: Check for the existance of the migrate tool with support for cockroach db
migrate-check-deps:
		@if [ -z `which migrate` ]; then \
				echo "[go get] installing golang-migrate cmd with cockroachdb support"; \
				echo "[go get] installing github.com/golang-migrate/migrate/v4/cmd/migrate"; \
				go install -tags 'cockroachdb postgres' -u github.com/golang-migrate/migrate/v4/cmd/migrate@latest;\
		fi

##	dsn_missing_error: Error string returned in an event where the CDB_DSN env var is missing / undefined
define dsn_missing_error

CDB_DSN envvar is undefined. To run migrations this envvar must point to a cockroach
db instance.For example, if you are running a local cockroachdb (with --insecure) and
have created a database called 'linkgraph' you can define the envvar by 
running:
	
export CDB_DSN='postgresql://root@localhost:26257/graphlink?sslmode=disable'

endef

export dsn_missing_error

##	check-cdb-env: Checks for the availability of the [CDB_DSN] env var and returns the [dsn_missing_error] if undefined
check-cdb-env:
ifndef CDB_DSN
		$(error ${dsn_missing_error})
endif

