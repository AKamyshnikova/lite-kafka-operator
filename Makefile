SHELL:=/bin/bash
PWD := $(shell pwd)
REPOSITORY?=docker-test-local.docker.mirantis.net
REPOSITORY_PATH?=tungsten-operator
NAME?=kafka-k8s-operator
VERSION?=$(shell hack/get_version.sh)
OPERATOR_IMAGE=$(REPOSITORY)/$(REPOSITORY_PATH)/$(NAME)
PUSHLATEST?=false
export GOPRIVATE=gerrit.mcp.mirantis.com/*


get-version: ##Get next possible version (see hack/get_version.sh)
	@echo ${VERSION}

# Get the currently used golang install path (in GOPATH/bin, unless GOBIN is set)
ifeq (,$(shell go env GOBIN))
GOBIN=$(shell go env GOPATH)/bin
else
GOBIN=$(shell go env GOBIN)
endif

all: manager

# Run tests
check: generate-k8s fmt vet crds test

# Build manager binary
manager: generate-k8s fmt vet
	go build -o bin/manager main.go

# Run against the configured Kubernetes cluster in ~/.kube/config
run: generate-k8s fmt vet crds
	go run ./main.go

# Install CRDs into a cluster
install: crds kustomize
	$(KUSTOMIZE) build config/crd | kubectl apply -f -

# Uninstall CRDs from a cluster
uninstall: crds kustomize
	$(KUSTOMIZE) build config/crd | kubectl delete -f -

# Deploy controller in the configured Kubernetes cluster in ~/.kube/config
deploy: crds kustomize
	cd config/manager && $(KUSTOMIZE) edit set image controller=${OPERATOR_IMAGE}:latest
	$(KUSTOMIZE) build config/default | kubectl apply -f -

# generate crd e.g. CRD, RBAC etc.
crds: controller-gen
	$(CONTROLLER_GEN) crd paths="./..." output:crd:artifacts:config=config/crd/bases

# Run go fmt against code
fmt:
	go fmt ./...

# Run go vet against code
vet:
	go vet ./...

# generate-k8s code
generate-k8s: controller-gen
	$(CONTROLLER_GEN) object:headerFile="hack/boilerplate.go.txt" paths="./..."

# Build the docker image
build: check
	docker build . -t $(OPERATOR_IMAGE):$(VERSION)
ifeq ($(PUSHLATEST), "true")
	docker tag $(OPERATOR_IMAGE):$(VERSION) $(OPERATOR_IMAGE):latest
endif

# Push the docker image
push:
	docker push $(OPERATOR_IMAGE):$(VERSION)
ifeq ($(PUSHLATEST), "true")
	docker push $(OPERATOR_IMAGE):latest
endif

image-path: ## Prints image path where it will be pushed
	@echo $(OPERATOR_IMAGE):$(VERSION)
image-version: ## Prints image version where it will be pushed
	@echo $(VERSION)

clean: ## Clean up the build artifacts
	@echo "Clean helm packages"
	rm -rf $(HELM_PACKAGE_DIR)/*tgz

tidy: check-git-config ## Update dependencies
	go mod tidy -v

lint:
	@if golangci-lint run -v ./...; then \
	  :; \
	else \
	  code=$$?; \
	  echo "Looks like golangci-lint failed. You can try autofixes with 'make fix'."; \
	  exit $$code; \
	fi

.PHONY: fix
fix:
	golangci-lint run -v --fix ./...

check-git-config: ## Check your git config
	@if ! git --no-pager config --get-regexp 'url\..*\.insteadof' 'https://gerrit.mcp.mirantis.com/a/' 1>/dev/null; then \
		echo "go get or go tidy may fail if you don't setup Git config."; \
		echo 'To set up Git to use SSH and SSH keys auth to access Gerrit you can run:'; \
        echo '	git config --global url."ssh://$${your_login}@gerrit.mcp.mirantis.com:29418/".insteadOf "https://gerrit.mcp.mirantis.com/a/"'; \
		echo 'where $${your_login} is your login in Gerrit'; \
	fi

update-git-config: ## Update your git config
	@if ! git --no-pager config --get-regexp 'url\..*\.insteadof' 'https://gerrit.mcp.mirantis.com/a/' 1>/dev/null; then \
		git config --global url."ssh://mcp-jenkins@gerrit.mcp.mirantis.com:29418/".insteadOf "https://gerrit.mcp.mirantis.com/a/"; \
	fi
# find or download controller-gen
# download controller-gen if necessary
controller-gen:
ifeq (, $(shell which controller-gen))
	@{ \
	set -e ;\
	CONTROLLER_GEN_TMP_DIR=$$(mktemp -d) ;\
	cd $$CONTROLLER_GEN_TMP_DIR ;\
	go mod init tmp ;\
	go get sigs.k8s.io/controller-tools/cmd/controller-gen@v0.6.2 ;\
	rm -rf $$CONTROLLER_GEN_TMP_DIR ;\
	}
CONTROLLER_GEN=$(GOBIN)/controller-gen
else
CONTROLLER_GEN=$(shell which controller-gen)
endif

kustomize:
ifeq (, $(shell which kustomize))
	@{ \
	set -e ;\
	KUSTOMIZE_GEN_TMP_DIR=$$(mktemp -d) ;\
	cd $$KUSTOMIZE_GEN_TMP_DIR ;\
	go mod init tmp ;\
	go get sigs.k8s.io/kustomize/kustomize/v3@v3.5.4 ;\
	rm -rf $$KUSTOMIZE_GEN_TMP_DIR ;\
	}
KUSTOMIZE=$(GOBIN)/kustomize
else
KUSTOMIZE=$(shell which kustomize)
endif

# generate bundle crd and metadata, then validate generated files.
bundle: crds
	operator-sdk generate kustomize crd -q
	kustomize build config/crd | operator-sdk generate bundle -q --overwrite --version $(VERSION) $(BUNDLE_METADATA_OPTS)
	operator-sdk bundle validate ./bundle

# Build the bundle image.
bundle-build:
	docker build -f bundle.Dockerfile -t $(BUNDLE_IMG) .

export CGO_ENABLED=0

test: ## Run tests. Using with param example: $ make type=cover test
ifeq ($(type),cover)
	    go test ./... -cover;
else ifeq ($(type),cover-html)
		go test ./... -coverprofile coverprofile.out 1> /dev/null;
		go tool cover -html coverprofile.out;
		@rm coverprofile.out;
else ifeq ($(type),cover-func)
		go test ./... -coverprofile coverprofile.out 1> /dev/null;
		go tool cover -func coverprofile.out;
		@rm coverprofile.out;
else ifeq ($(type),cover-func-no-zero)
		go test ./... -coverprofile coverprofile.out 1> /dev/null;
		go tool cover -func coverprofile.out | grep -v -e "\t0.0%";
		@rm coverprofile.out;
else
	    go test ./...
endif

.PHONY: help
help: ## Display this help
	@echo -e "Usage:\n  make \033[36m<target>\033[0m"
	@awk 'BEGIN {FS = ":.*##"}; \
		/^[a-zA-Z0-9_-]+:.*?##/ { printf "  \033[36m%-15s\033[0m %s\n", $$1, $$2 } \
		/^##@/ { printf "\n\033[1m%s\033[0m\n", substr($$0, 5) } ' $(MAKEFILE_LIST)