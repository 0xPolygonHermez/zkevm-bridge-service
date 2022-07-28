DOCKERCOMPOSE := docker-compose -f docker-compose.yml
DOCKERCOMPOSEDBCORE := hez-postgres-core
DOCKERCOMPOSEDBBRIDGE := hez-postgres-bridge
DOCKERCOMPOSEHERMEZCORE := hez-core
DOCKERCOMPOSENETWORK := hez-network
DOCKERCOMPOSEPROVER := hez-prover
DOCKERCOMPOSEBRIDGE := hez-bridge
DOCKERCOMPOSEMOCKSERVER := hez-bridge-mock

RUNDBCORE := $(DOCKERCOMPOSE) up -d $(DOCKERCOMPOSEDBCORE)
RUNDBBRIDGE := $(DOCKERCOMPOSE) up -d $(DOCKERCOMPOSEDBBRIDGE)
RUNDBS := ${RUNDBCORE} && ${RUNDBBRIDGE}
RUNCORE := $(DOCKERCOMPOSE) up -d $(DOCKERCOMPOSEHERMEZCORE)
RUNNETWORK := $(DOCKERCOMPOSE) up -d $(DOCKERCOMPOSENETWORK)
RUNPROVER := $(DOCKERCOMPOSE) up -d $(DOCKERCOMPOSEPROVER)
RUNBRIDGE := $(DOCKERCOMPOSE) up -d $(DOCKERCOMPOSEBRIDGE)
RUNMOCKBRIDGE := $(DOCKERCOMPOSE) up -d $(DOCKERCOMPOSEMOCKSERVER)
RUN := $(DOCKERCOMPOSE) up -d

STOPDBCORE := $(DOCKERCOMPOSE) stop $(DOCKERCOMPOSEDBCORE) && $(DOCKERCOMPOSE) rm -f $(DOCKERCOMPOSEDBCORE)
STOPDBBRIDGE := $(DOCKERCOMPOSE) stop $(DOCKERCOMPOSEDBBRIDGE) && $(DOCKERCOMPOSE) rm -f $(DOCKERCOMPOSEDBBRIDGE)
STOPDBS := ${STOPDBCORE} && ${STOPDBBRIDGE}
STOPCORE := $(DOCKERCOMPOSE) stop $(DOCKERCOMPOSEHERMEZCORE) && $(DOCKERCOMPOSE) rm -f $(DOCKERCOMPOSEHERMEZCORE)
STOPNETWORK := $(DOCKERCOMPOSE) stop $(DOCKERCOMPOSENETWORK) && $(DOCKERCOMPOSE) rm -f $(DOCKERCOMPOSENETWORK)
STOPPROVER := $(DOCKERCOMPOSE) stop $(DOCKERCOMPOSEPROVER) && $(DOCKERCOMPOSE) rm -f $(DOCKERCOMPOSEPROVER)
STOPBRIDGE := $(DOCKERCOMPOSE) stop $(DOCKERCOMPOSEBRIDGE) && $(DOCKERCOMPOSE) rm -f $(DOCKERCOMPOSEBRIDGE)
STOPMOCKBRIDGE := $(DOCKERCOMPOSE) stop $(DOCKERCOMPOSEMOCKSERVER) && $(DOCKERCOMPOSE) rm -f $(DOCKERCOMPOSEMOCKSERVER)
STOP := $(DOCKERCOMPOSE) down --remove-orphans

VERSION := $(shell git describe --tags --always)
COMMIT := $(shell git rev-parse --short HEAD)
DATE := $(shell date +%Y-%m-%dT%H:%M:%S%z)
LDFLAGS := -ldflags "-X main.version=$(VERSION) -X main.commit=$(COMMIT) -X main.date=$(DATE)"

GOBASE := $(shell pwd)
GOBIN := $(GOBASE)/dist
GOENVVARS := GOBIN=$(GOBIN)
GOBINARY := hezbridge
GOCMD := $(GOBASE)/cmd

LINT := $$(go env GOPATH)/bin/golangci-lint run --timeout=5m -E whitespace -E gosec -E gci -E misspell -E gomnd -E gofmt -E goimports -E golint --exclude-use-default=false --max-same-issues 0
BUILD := $(GOENVVARS) go build $(LDFLAGS) -o $(GOBIN)/$(GOBINARY) $(GOCMD)

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
	$(STOPDBBRIDGE) || true
	$(RUNDBBRIDGE); sleep 5
	trap '$(STOPDBBRIDGE)' EXIT; go test --cover -short -p 1 ./...

.PHONY: install-linter
install-linter: ## Installs the linter
	curl -sSfL https://raw.githubusercontent.com/golangci/golangci-lint/master/install.sh | sh -s -- -b $$(go env GOPATH)/bin v1.46.2

.PHONY: build-docker
build-docker: ## Builds a docker image with the core binary
	docker build -t hermeznetwork/hermez-bridge -f ./Dockerfile .

.PHONY: run-db-core
run-db-core: ## Runs the node database
	$(RUNDBCORE)

.PHONY: stop-db-core
stop-db-core: ## Stops the node database
	$(STOPDBCORE)

.PHONY: run-db-bridge
run-db-bridge: ## Runs the node database
	$(RUNDBBRIDGE)

.PHONY: stop-db-bridge
stop-db-bridge: ## Stops the node database
	$(STOPDBBRIDGE)

.PHONY: run-dbs
run-dbs: ## Runs the node database
	$(RUNDBS)

.PHONY: stop-dbs
stop-dbs: ## Stops the node database
	$(STOPDBS)

.PHONY: run-core
run-core: ## Runs the core
	$(RUNCORE)

.PHONY: stop-core
stop-core: ## Stops the core
	$(STOPCORE)

.PHONY: run-network
run-network: ## Runs the l1 network
	$(RUNNETWORK)

.PHONY: stop-network
stop-network: ## Stops the l1 network
	$(STOPNETWORK)

.PHONY: run-prover
run-prover: ## Runs the zk prover
	$(RUNPROVER)

.PHONY: stop-prover
stop-prover: ## Stops the zk prover
	$(STOPPROVER)

.PHONY: run-bridge
run-bridge: ## Runs the bridge service
	$(RUNBRIDGE)

.PHONY: stop-bridge
stop-bridge: ## Stops the bridge service
	$(STOPBRIDGE)

.PHONY: stop
stop: ## Stops all services
	$(STOP)

.PHONY: restart
restart: stop run ## Executes `make stop` and `make run` commands

.PHONY: run
run: ## runs all services
	$(RUN)

.PHONY: run-mockserver
run-mockserver: ## runs the mocked restful server
	$(RUNDBBRIDGE)
	sleep 3
	$(RUNMOCKBRIDGE)

.PHONY: update-external-dependencies
update-external-dependencies: ## Updates external dependencies like images, test vectors or proto files
	go run ./scripts/cmd/... updatedeps

.PHONY: generate-code-from-proto
generate-code-from-proto:
	cd proto/src/proto/bridge/v1 && protoc --proto_path=. --proto_path=../../../../../third_party --go_out=../../../../../bridgectrl/pb --go-grpc_out=../../../../../bridgectrl/pb  --go_opt=paths=source_relative --go-grpc_opt=paths=source_relative query.proto
	cd proto/src/proto/bridge/v1 && protoc --proto_path=. --proto_path=../../../../../third_party --grpc-gateway_out=logtostderr=true:../../../../../bridgectrl/pb --grpc-gateway_opt=paths=source_relative query.proto

.PHONY: stop-mockserver
stop-mockserver: ## Stops the mock bridge service
	$(STOPMOCKBRIDGE)

.PHONY: performance-test
performance-test: ## Performance test of rest api and db transaction
	go run ./test/performance/... 1000

.PHONY: test-full
test-full: build-docker ## Runs all tests checking race conditions
	$(STOPDBS)
	$(RUNDBS); sleep 7
	trap '$(STOPDBS)' EXIT; MallocNanoZone=0 go test -race -p 1 -timeout 2400s ./...

.PHONY: validate
validate: lint build test-full ## Validates the whole integrity of the code base

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
	mockery --name=BroadcastServiceClient --srcpkg=github.com/0xPolygonHermez/zkevm-node/sequencer/broadcast/pb --output=synchronizer --outpkg=synchronizer --structname=grpcMock --filename=mock_grpc.go
