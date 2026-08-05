SHELL := /bin/bash
export GO111MODULE = on

GO      := go
GOPATH  ?= $(shell $(GO) env GOPATH)
GOCMD   := $(GOPATH)/bin

CONTROLLER_GEN_VERSION := v0.16.1
KUSTOMIZE_VERSION      := v5.4.2

API_DIR      := api
CONFIG_DIR   := config
CHARTS_DIR   := helm/cloudorch
BIN_DIR      := $(abspath bin)

OPERATOR     := cloudorch

IMAGE_NAME   := cloudorch
IMAGE_TAG    ?= latest

$(BIN_DIR):
	mkdir -p $(BIN_DIR)

.PHONY: tools
tools: $(BIN_DIR)
	GOBIN=$(BIN_DIR) $(GO) install sigs.k8s.io/controller-tools/cmd/controller-gen@$(CONTROLLER_GEN_VERSION)
	GOBIN=$(BIN_DIR) $(GO) install sigs.k8s.io/kustomize/kustomize/v5@$(KUSTOMIZE_VERSION)

.PHONY: generate
generate: tools
	$(BIN_DIR)/controller-gen object:headerFile=hack/boilerplate.go.txt paths=./api/...
	$(BIN_DIR)/controller-gen object:headerFile=hack/boilerplate.go.txt paths=./internal/...

.PHONY: manifests
manifests: tools
	mkdir -p $(CONFIG_DIR)/crd/bases $(CONFIG_DIR)/rbac $(CONFIG_DIR)/webhook
	$(BIN_DIR)/controller-gen crd:crdVersions=v1 paths=./api/... output:crd:artifacts:config=$(CONFIG_DIR)/crd/bases
	$(BIN_DIR)/controller-gen rbac:roleName=manager-role paths=./... output:rbac:artifacts:config=$(CONFIG_DIR)/rbac
	$(BIN_DIR)/controller-gen webhook paths=./... output:webhook:artifacts:config=$(CONFIG_DIR)/webhook

.PHONY: build
build: generate
	CGO_ENABLED=0 $(GO) build -o $(BIN_DIR)/$(OPERATOR) ./main.go

.PHONY: run
run: build
	$(BIN_DIR)/$(OPERATOR)

.PHONY: docker-build
docker-build:
	docker build -t $(IMAGE_NAME):$(IMAGE_TAG) -f build/Dockerfile .

.PHONY: docker-push
docker-push: docker-build
	docker push $(IMAGE_NAME):$(IMAGE_TAG)

.PHONY: helm-template
helm-template:
	helm template $(OPERATOR) $(CHARTS_DIR) --set image.tag=$(IMAGE_TAG) --set image.repository=$(IMAGE_NAME)

.PHONY: helm-install
helm-install:
	helm upgrade --install $(OPERATOR) $(CHARTS_DIR) --namespace cloudorch-system --create-namespace --set image.tag=$(IMAGE_TAG) --set image.repository=$(IMAGE_NAME)

.PHONY: test
test: generate
	$(GO) test -v -race -coverprofile=coverage.out ./internal/...
	$(GO) tool cover -html=coverage.out -o coverage.html

.PHONY: test-integration
test-integration:
	$(GO) test -v -tags=integration ./test/envtest/...

.PHONY: test-policy
test-policy:
	conftest test $(CONFIG_DIR)/policies/ --policy $(CONFIG_DIR)/policies

.PHONY: lint
lint:
	golangci-lint run ./...

.PHONY: deploy
deploy: manifests
	kubectl apply -f $(CONFIG_DIR)/crd/bases
	kubectl apply -f $(CONFIG_DIR)/rbac
	kubectl apply -f $(CONFIG_DIR)/webhook
	kubectl apply -f $(CHARTS_DIR)/crds
	helm upgrade --install $(OPERATOR) $(CHARTS_DIR) --namespace cloudorch-system --create-namespace

.PHONY: undeploy
undeploy:
	helm uninstall $(OPERATOR) --namespace cloudorch-system || true
	kubectl delete -f $(CONFIG_DIR)/webhook || true
	kubectl delete -f $(CONFIG_DIR)/rbac || true
	kubectl delete -f $(CONFIG_DIR)/crd/bases || true

.PHONY: clean
clean:
	rm -rf $(BIN_DIR)
	rm -f coverage.out coverage.html
	rm -rf dist/

.PHONY: fmt
fmt:
	$(GO) fmt ./...

.PHONY: vet
vet:
	$(GO) vet ./...

.PHONY: all
all: clean fmt vet generate manifests build test