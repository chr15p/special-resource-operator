SPECIALRESOURCE  ?= driver-container-base
NAMESPACE        ?= openshift-special-resource-operator
PULLPOLICY       ?= IfNotPresent
TAG              ?= $(shell git branch --show-current)
IMAGE            ?= quay.io/openshift-psap/special-resource-operator:$(TAG)
CSPLIT           ?= csplit - --prefix="" --suppress-matched --suffix-format="%04d.yaml"  /---/ '{*}' --silent
YAMLFILES        ?= $(shell  find manifests-gen config/recipes -name "*.yaml"  -not \( -path "config/recipes/lustre-client/*" -prune \) )

export PATH := go/bin:$(PATH)
include Makefile.specialresource.mk
include Makefile.helper.mk


patch:
	git diff vendor/github.com/go-logr/zapr/zapr.go > /tmp/zapr.patch
	git apply /tmp/zapr.patch


helm-lint:
	helm lint -f charts/global/values.yaml \
		charts/example/*               \
		charts/lustre/*

kube-lint: kube-linter
	$(KUBELINTER) lint $(YAMLFILES)

lint: golangci-lint
	$(GOLANGCILINT) run -v --timeout 5m0s

verify: fmt vet
unit:
	@echo "TODO UNIT TEST"

go-deploy-manifests: manifests
	go run test/deploy/deploy.go -path ./manifests

go-undeploy-manifests:
	go run test/undeploy/undeploy.go -path ./manifests

test-e2e-upgrade: go-deploy-manifests

test-e2e:
	for d in basic; do \
          KUBERNETES_CONFIG="$(KUBECONFIG)" go test -v -timeout 40m ./test/e2e/$$d -ginkgo.v -ginkgo.noColor -ginkgo.failFast || exit; \
        done

# Current Operator version
VERSION ?= 0.0.1
# Default bundle image tag
BUNDLE_IMG ?= sro-bundle:$(VERSION)
# Options for 'bundle-build'
ifneq ($(origin CHANNELS), undefined)
BUNDLE_CHANNELS := --channels=$(CHANNELS)
endif
ifneq ($(origin DEFAULT_CHANNEL), undefined)
BUNDLE_DEFAULT_CHANNEL := --default-channel=$(DEFAULT_CHANNEL)
endif
BUNDLE_METADATA_OPTS ?= $(BUNDLE_CHANNELS) $(BUNDLE_DEFAULT_CHANNEL)

# Produce CRDs that work back to Kubernetes 1.11 (no version conversion)
CRD_OPTIONS ?= "crd:crdVersions=v1,trivialVersions=true"

# Get the currently used golang install path (in GOPATH/bin, unless GOBIN is set)
ifeq (,$(shell go env GOBIN))
GOBIN=$(shell go env GOPATH)/bin
else
GOBIN=$(shell go env GOBIN)
endif

# GENERATED all: manager
all: $(SPECIALRESOURCE)

# Run tests
test: # generate fmt vet manifests-gen
	go test ./... -coverprofile cover.out

# Build manager binary
manager: generate fmt vet
	go build -mod=vendor -o /tmp/bin/manager main.go

# Run against the configured Kubernetes cluster in ~/.kube/config
run: generate fmt vet manifests-gen
	go run -mod=vendor ./main.go

configure:
	# TODO kustomize cannot set name of namespace according to settings, hack TODO
	cd config/namespace && sed -i 's/name: .*/name: $(NAMESPACE)/g' namespace.yaml
	cd config/namespace && $(KUSTOMIZE) edit set namespace $(NAMESPACE)
	cd config/default && $(KUSTOMIZE) edit set namespace $(NAMESPACE)
	cd config/manager && $(KUSTOMIZE) edit set image controller=$(IMAGE)


# Deploy controller in the configured Kubernetes cluster in ~/.kube/config
deploy: manifests
	$(KUSTOMIZE) build config/namespace | kubectl apply -f -
	$(shell sleep 5)
	$(KUSTOMIZE) build config/cr | kubectl apply -f -


# If the CRD is deleted before the CRs the CRD finalizer will hang forever
# The specialresource finalizer will not execute either
undeploy: kustomize
	if [ ! -z "$$(kubectl get crd | grep specialresource)" ]; then                     \
		kubectl delete --ignore-not-found specialresource --all --all-namespaces; \
	fi;
	$(KUSTOMIZE) build config/namespace | kubectl delete --ignore-not-found -f -


# Generate manifests-gen e.g. CRD, RBAC etc.
manifests-gen: controller-gen
	$(CONTROLLER_GEN) $(CRD_OPTIONS) rbac:roleName=manager-role webhook paths="./..." output:crd:artifacts:config=config/crd/bases

manifests: manifests-gen kustomize configure
	cd $@; $(KUSTOMIZE) build ../config/namespace | $(CSPLIT)
	cd $@; bash ../scripts/rename.sh
	cd $@; $(KUSTOMIZE) build ../config/cr > 0015_specialresource_special-resource-preamble.yaml

# Run go fmt against code
fmt:
	go fmt ./...

# Run go vet against code
vet:
	go vet --mod=vendor ./...

# Generate code
generate: controller-gen
	$(CONTROLLER_GEN) object:headerFile="hack/boilerplate.go.txt" paths="./..."

# Build the docker image
local-image-build: helm-lint test generate manifests-gen
	podman build -f Dockerfile.ubi8 --no-cache . -t $(IMAGE)

# Push the docker image
local-image-push:
	podman push $(IMAGE)


# Generate bundle manifests-gen and metadata, then validate generated files.
.PHONY: bundle
bundle: manifests-gen
	operator-sdk generate kustomize manifests-gen -q
	cd config/manager && $(KUSTOMIZE) edit set image controller=$(IMAGE)
	$(KUSTOMIZE) build config/manifests-gen | operator-sdk generate bundle -q --overwrite --version $(VERSION) $(BUNDLE_METADATA_OPTS)
	operator-sdk bundle validate ./bundle

# Build the bundle image.
.PHONY: bundle-build
bundle-build:
	podman build -f bundle.Dockerfile -t $(BUNDLE_IMG) .
