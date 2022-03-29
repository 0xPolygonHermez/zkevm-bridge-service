DOCKERCOMPOSE := docker-compose -f docker-compose.yml
DOCKERCOMPOSEDBCORE := hez-postgres-core
DOCKERCOMPOSEDBBRIDGE := hez-postgres-bridge
DOCKERCOMPOSEHERMEZCORE := hez-core
DOCKERCOMPOSENETWORK := hez-network
DOCKERCOMPOSEPROVER := hez-prover
DOCKERCOMPOSEBRIDGE := hez-bridge
DOCKERCOMPOSEMOCKSERVER := hez-bridge-mock

RUNDBCORE := $(DOCKERCOMPOSE) up -d $(DOCKERCOMPOSEDBCORE)
RUNDBBRDIGE := $(DOCKERCOMPOSE) up -d $(DOCKERCOMPOSEDBBRIDGE)
RUNDBS := ${RUNDBCORE} && ${RUNDBBRDIGE}
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
	$(RUNDBBRDIGE); sleep 5
	trap '$(STOPDBBRIDGE)' EXIT; go test --cover -short -p 1 ./...

.PHONY: install-linter
install-linter: ## Installs the linter
	curl -sSfL https://raw.githubusercontent.com/golangci/golangci-lint/master/install.sh | sh -s -- -b $$(go env GOPATH)/bin v1.39.0

.PHONY: build-docker
build-docker: ## Builds a docker image with the core binary
	docker build -t hezbridge -f ./Dockerfile . --build-arg PRIVATE_TOKEN=${PRIVATE_TOKEN}

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
	$(RUNDBBRDIGE)
	sleep 3
	$(RUNMOCKBRIDGE)

.PHONY: proto-gen
proto-gen:
	protoc --proto_path=proto/hermez/bridge/v1 --proto_path=third_party --go_out=bridgetree/pb --go-grpc_out=bridgetree/pb  --go_opt=paths=source_relative --go-grpc_opt=paths=source_relative query.proto
	protoc --proto_path=proto/hermez/bridge/v1 --proto_path=third_party --grpc-gateway_out=logtostderr=true:bridgetree/pb --grpc-gateway_opt=paths=source_relative query.proto