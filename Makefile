include version.mk

DOCKER_COMPOSE := docker-compose -f docker-compose.yml
DOCKER_COMPOSE_STATE_DB := zkevm-state-db
DOCKER_COMPOSE_POOL_DB := zkevm-pool-db
DOCKER_COMPOSE_RPC_DB := zkevm-rpc-db
DOCKER_COMPOSE_BRIDGE_DB := zkevm-bridge-db
DOCKER_COMPOSE_ZKEVM_NODE := zkevm-node
DOCKER_COMPOSE_L1_NETWORK := zkevm-mock-l1-network
DOCKER_COMPOSE_ZKPROVER := zkevm-prover
DOCKER_COMPOSE_BRIDGE := zkevm-bridge-service

RUN_STATE_DB := $(DOCKER_COMPOSE) up -d $(DOCKER_COMPOSE_STATE_DB)
RUN_POOL_DB := $(DOCKER_COMPOSE) up -d $(DOCKER_COMPOSE_POOL_DB)
RUN_BRIDGE_DB := $(DOCKER_COMPOSE) up -d $(DOCKER_COMPOSE_BRIDGE_DB)
RUN_DBS := ${RUN_BRIDGE_DB} && ${RUN_STATE_DB} && ${RUN_POOL_DB}
RUN_NODE := $(DOCKER_COMPOSE) up -d $(DOCKER_COMPOSE_ZKEVM_NODE)
RUN_L1_NETWORK := $(DOCKER_COMPOSE) up -d $(DOCKER_COMPOSE_L1_NETWORK)
RUN_ZKPROVER := $(DOCKER_COMPOSE) up -d $(DOCKER_COMPOSE_ZKPROVER)
RUN_BRIDGE := $(DOCKER_COMPOSE) up -d $(DOCKER_COMPOSE_BRIDGE)

STOP_NODE_DB := $(DOCKER_COMPOSE) stop $(DOCKER_COMPOSE_NODE_DB) && $(DOCKER_COMPOSE) rm -f $(DOCKER_COMPOSE_NODE_DB)
STOP_BRIDGE_DB := $(DOCKER_COMPOSE) stop $(DOCKER_COMPOSE_BRIDGE_DB) && $(DOCKER_COMPOSE) rm -f $(DOCKER_COMPOSE_BRIDGE_DB)
STOP_DBS := ${STOP_NODE_DB} && ${STOP_BRIDGE_DB}
STOP_NODE := $(DOCKER_COMPOSE) stop $(DOCKER_COMPOSE_ZKEVM_NODE) && $(DOCKER_COMPOSE) rm -f $(DOCKER_COMPOSE_ZKEVM_NODE)
STOP_NETWORK := $(DOCKER_COMPOSE) stop $(DOCKER_COMPOSE_L1_NETWORK) && $(DOCKER_COMPOSE) rm -f $(DOCKER_COMPOSE_L1_NETWORK)
STOP_ZKPROVER := $(DOCKER_COMPOSE) stop $(DOCKER_COMPOSE_ZKPROVER) && $(DOCKER_COMPOSE) rm -f $(DOCKER_COMPOSE_ZKPROVER)
STOP_BRIDGE := $(DOCKER_COMPOSE) stop $(DOCKER_COMPOSE_BRIDGE) && $(DOCKER_COMPOSE) rm -f $(DOCKER_COMPOSE_BRIDGE)
STOP := $(DOCKER_COMPOSE) down --remove-orphans

LDFLAGS += -X 'github.com/0xPolygonHermez/zkevm-bridge-service.Version=$(VERSION)'
LDFLAGS += -X 'github.com/0xPolygonHermez/zkevm-bridge-service.GitRev=$(GITREV)'
LDFLAGS += -X 'github.com/0xPolygonHermez/zkevm-bridge-service.GitBranch=$(GITBRANCH)'
LDFLAGS += -X 'github.com/0xPolygonHermez/zkevm-bridge-service.BuildDate=$(DATE)'

GO_BASE := $(shell pwd)
GO_BIN := $(GO_BASE)/dist
GO_ENV_VARS := GO_BIN=$(GO_BIN)
GO_BINARY := zkevm-bridge
GO_CMD := $(GO_BASE)/cmd

LINT := $$(go env GOPATH)/bin/golangci-lint run --timeout=5m -E whitespace -E gosec -E gci -E misspell -E gomnd -E gofmt -E goimports --exclude-use-default=false --max-same-issues 0
BUILD := $(GO_ENV_VARS) go build -ldflags "all=$(LDFLAGS)" -o $(GO_BIN)/$(GO_BINARY) $(GO_CMD)

.PHONY: build
build: ## Build the binary locally into ./dist
	$(BUILD)

.PHONY: lint
lint: ## runs linter
	$(LINT)

.PHONY: install-git-hooks
install-git-hooks: ## Moves hook files to the .git/hooks directory
	cp .github/hooks/* .git/hooks

.PHONY: test
test: ## Runs only short tests without checking race conditions
	$(STOP_BRIDGE_DB) || true
	$(RUN_BRIDGE_DB); sleep 3
	trap '$(STOP_BRIDGE_DB)' EXIT; go test --cover -short -p 1 ./...

.PHONY: install-linter
install-linter: ## Installs the linter
	curl -sSfL https://raw.githubusercontent.com/golangci/golangci-lint/master/install.sh | sh -s -- -b $$(go env GOPATH)/bin v1.51.2

.PHONY: build-docker
build-docker: ## Builds a docker image with the zkevm bridge binary
	docker build -t zkevm-bridge-service -f ./Dockerfile .

.PHONY: run-db-node
run-db-node: ## Runs the node database
	$(RUN_NODE_DB)

.PHONY: stop-db-node
stop-db-node: ## Stops the node database
	$(STOP_NODE_DB)

.PHONY: run-db-bridge
run-db-bridge: ## Runs the node database
	$(RUN_BRIDGE_DB)

.PHONY: stop-db-bridge
stop-db-bridge: ## Stops the node database
	$(STOP_BRIDGE_DB)

.PHONY: run-dbs
run-dbs: ## Runs the node database
	$(RUN_DBS)

.PHONY: stop-dbs
stop-dbs: ## Stops the node database
	$(STOP_DBS)

.PHONY: run-node
run-node: ## Runs the node
	$(RUN_NODE)

.PHONY: stop-node
stop-node: ## Stops the node
	$(STOP_NODE)

.PHONY: run-network
run-network: ## Runs the l1 network
	$(RUN_L1_NETWORK)

.PHONY: stop-network
stop-network: ## Stops the l1 network
	$(STOP_NETWORK)

.PHONY: run-prover
run-prover: ## Runs the zk prover
	$(RUN_ZKPROVER)

.PHONY: stop-prover
stop-prover: ## Stops the zk prover
	$(STOP_ZKPROVER)

.PHONY: run-bridge
run-bridge: ## Runs the bridge service
	$(RUN_BRIDGE)

.PHONY: stop-bridge
stop-bridge: ## Stops the bridge service
	$(STOP_BRIDGE)

.PHONY: stop
stop: ## Stops all services
	$(STOP)

.PHONY: restart
restart: stop run ## Executes `make stop` and `make run` commands

.PHONY: run
run: stop ## runs all services
	$(RUN_DBS)
	$(RUN_L1_NETWORK)
	sleep 5
	$(RUN_ZKPROVER)
	sleep 3
	$(RUN_NODE)
	sleep 7
	$(RUN_BRIDGE)

.PHONY: update-external-dependencies
update-external-dependencies: ## Updates external dependencies like images, test vectors or proto files
	go run ./scripts/cmd/... updatedeps

.PHONY: generate-code-from-proto
generate-code-from-proto:
	cd proto/src/proto/bridge/v1 && protoc --proto_path=. --proto_path=../../../../../third_party --go_out=../../../../../bridgectrl/pb --go-grpc_out=../../../../../bridgectrl/pb  --go_opt=paths=source_relative --go-grpc_opt=paths=source_relative query.proto
	cd proto/src/proto/bridge/v1 && protoc --proto_path=. --proto_path=../../../../../third_party --grpc-gateway_out=logtostderr=true:../../../../../bridgectrl/pb --grpc-gateway_opt=paths=source_relative query.proto

.PHONY: stop-mockserver
stop-mockserver: ## Stops the mock bridge service
	$(STOP_BRIDGE_MOCK)

.PHONY: bench
bench: ## benchmark test
	$(STOP_BRIDGE_DB) || true
	$(RUN_BRIDGE_DB); sleep 3
	trap '$(STOP_BRIDGE_DB)' EXIT; go test -run=NOTEST -timeout=30m -bench=Small ./test/benchmark/...

.PHONY: bench-full
bench-full: export ZKEVM_BRIDGE_DATABASE_PORT = 5432
bench-full: ## benchmark full test
	cd test/benchmark && \
	go test -run=NOTEST -bench=Small . && \
	go test -run=NOTEST -bench=Medium . && \
	go test -run=NOTEST -timeout=30m -bench=Large .

.PHONY: test-full
test-full: build-docker stop run ## Runs all tests checking race conditions
	sleep 3
	trap '$(STOP)' EXIT; MallocNanoZone=0 go test -race -p 1 -timeout 2400s ./test/e2e/... -count 1 -tags='e2e'

.PHONY: test-edge
test-edge: build-docker stop run ## Runs all tests checking race conditions
	sleep 3
	trap '$(STOP)' EXIT; MallocNanoZone=0 go test -race -p 1 -timeout 2400s ./test/e2e/... -count 1 -tags='edge'

.PHONY: validate
validate: lint build test-full ## Validates the whole integrity of the code base

## Help display.
## Pulls comments from beside commands and prints a nicely formatted
## display with the commands and their usage information.
.DEFAULT_GOAL := help

.PHONY: help
help: ## Prints this help
		@grep -h -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) \
		| sort \
		| awk 'BEGIN {FS = ":.*?## "}; {printf "\033[36m%-30s\033[0m %s\n", $$1, $$2}'

.PHONY: generate-mocks
generate-mocks: ## Generates mocks for the tests, using mockery tool
	mockery --name=ethermanInterface --dir=synchronizer --output=synchronizer --outpkg=synchronizer --structname=ethermanMock --filename=mock_etherman.go
	mockery --name=storageInterface --dir=synchronizer --output=synchronizer --outpkg=synchronizer --structname=storageMock --filename=mock_storage.go
	mockery --name=bridgectrlInterface --dir=synchronizer --output=synchronizer --outpkg=synchronizer --structname=bridgectrlMock --filename=mock_bridgectrl.go
	mockery --name=Tx --srcpkg=github.com/jackc/pgx/v4 --output=synchronizer --outpkg=synchronizer --structname=dbTxMock --filename=mock_dbtx.go
	mockery --name=zkEVMClientInterface --dir=synchronizer --output=synchronizer --outpkg=synchronizer --structname=zkEVMClientMock --filename=mock_zkevmclient.go
