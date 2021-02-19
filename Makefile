export CGO_ENABLED=1
export GO111MODULE=on

.PHONY: build

ONOS_KPIMON_VERSION := latest
ONOS_BUILD_VERSION := v0.6.6
ONOS_PROTOC_VERSION := v0.6.6
BUF_VERSION := 0.27.1

build: # @HELP build the Go binaries and run all validations (default)
build:
	GOPRIVATE="github.com/onosproject/*" go build -o build/_output/onos-kpimon ./cmd/onos-kpimon

test: # @HELP run the unit tests and source code validation
test: build deps linters license_check
	go test -race github.com/onosproject/onos-kpimon/pkg/...
	go test -race github.com/onosproject/onos-kpimon/cmd/...

jenkins-test:  # @HELP run the unit tests and source code validation producing a junit style report for Jenkins
jenkins-test: build-tools deps license_check linters
	TEST_PACKAGES=github.com/onosproject/onos-kpimon/... ./../build-tools/build/jenkins/make-unit

coverage: # @HELP generate unit test coverage data
coverage: build deps linters license_check
	./build/bin/coveralls-coverage

deps: # @HELP ensure that the required dependencies are in place
	GOPRIVATE="github.com/onosproject/*" go build -v ./...
	bash -c "diff -u <(echo -n) <(git diff go.mod)"
	bash -c "diff -u <(echo -n) <(git diff go.sum)"

linters: golang-ci # @HELP examines Go source code and reports coding problems
	golangci-lint run --timeout 5m

build-tools: # @HELP install the ONOS build tools if needed
	@if [ ! -d "../build-tools" ]; then cd .. && git clone https://github.com/onosproject/build-tools.git; fi

jenkins-tools: # @HELP installs tooling needed for Jenkins
	cd .. && go get -u github.com/jstemmer/go-junit-report && go get github.com/t-yuki/gocover-cobertura

golang-ci: # @HELP install golang-ci if not present
	golangci-lint --version || curl -sfL https://install.goreleaser.com/github.com/golangci/golangci-lint.sh | sh -s -- -b `go env GOPATH`/bin v1.36.0

license_check: build-tools # @HELP examine and ensure license headers exist
	./../build-tools/licensing/boilerplate.py -v --rootdir=${CURDIR} --boilerplate LicenseRef-ONF-Member-1.0

gofmt: # @HELP run the Go format validation
	bash -c "diff -u <(echo -n) <(gofmt -d pkg/ cmd/ tests/)"

buflint: #@HELP run the "buf check lint" command on the proto files in 'api'
	docker run -it -v `pwd`:/go/src/github.com/onosproject/onos-kpimon \
		-w /go/src/github.com/onosproject/onos-kpimon/api \
		bufbuild/buf:${BUF_VERSION} check lint

protos: # @HELP compile the protobuf files (using protoc-go Docker)
protos:
	docker run -it -v `pwd`:/go/src/github.com/onosproject/onos-kpimon \
		-w /go/src/github.com/onosproject/onos-kpimon \
		--entrypoint build/bin/compile-protos.sh \
		onosproject/protoc-go:${ONOS_PROTOC_VERSION}

onos-kpimon-base-docker: # @HELP build onos-kpimon base Docker image
	docker build . -f build/base/Dockerfile \
		--build-arg ONOS_BUILD_VERSION=${ONOS_BUILD_VERSION} \
		--build-arg ONOS_MAKE_TARGET=build \
		-t onosproject/onos-kpimon-base:${ONOS_KPIMON_VERSION}

onos-kpimon-docker: # @HELP build onos-kpimon Docker image
onos-kpimon-docker: onos-kpimon-base-docker
	docker build . -f build/onos-kpimon/Dockerfile \
		--build-arg ONOS_KPIMON_BASE_VERSION=${ONOS_KPIMON_VERSION} \
		-t onosproject/onos-kpimon:${ONOS_KPIMON_VERSION}

images: # @HELP build all Docker images
images: build onos-kpimon-docker

kind: # @HELP build Docker images and add them to the currently configured kind cluster
kind: images
	@if [ "`kind get clusters`" = '' ]; then echo "no kind cluster found" && exit 1; fi
	kind load docker-image onosproject/onos-kpimon:${ONOS_KPIMON_VERSION}

all: build images

publish: # @HELP publish version on github and dockerhub
	./../build-tools/publish-version ${VERSION} onosproject/onos-kpimon

jenkins-publish: build-tools jenkins-tools # @HELP Jenkins calls this to publish artifacts
	../build-tools/release-merge-commit

bumponosdeps: # @HELP update "onosproject" go dependencies and push patch to git. Add a version to dependency to make it different to $VERSION
	./../build-tools/bump-onos-deps ${VERSION}

clean: # @HELP remove all the build artifacts
	rm -rf ./build/_output ./vendor ./cmd/onos-kpimon/onos-kpimon ./cmd/onos/onos
	go clean -testcache github.com/onosproject/onos-kpimon/...

help:
	@grep -E '^.*: *# *@HELP' $(MAKEFILE_LIST) \
    | sort \
    | awk ' \
        BEGIN {FS = ": *# *@HELP"}; \
        {printf "\033[36m%-30s\033[0m %s\n", $$1, $$2}; \
    '
