APP ?= fix_log4j2
MODULE := github.com/YunDingLab/${APP}

GO ?= go
VERSION ?= $(shell git describe --tags)
GIT_COMMIT = $(shell git rev-parse --short HEAD || echo unsupported)
GO_VERSION = $(shell go version)
BUILD_TIME = $(shell date "+%Y-%m-%d_%H:%M:%S")
IMAGE_GROUP ?= ccr.ccs.tencentyun.com/yunding
IMAGE_NAME = ${IMAGE_GROUP}/${APP}
IMAGE_TEST = ${IMAGE_NAME}:${GIT_COMMIT}
IMAGE_RELEASE = ${IMAGE_NAME}:${VERSION}

LD_FLAGS = -X ${MODULE}/version.version=$(VERSION) \
 -X ${MODULE}/version.gitCommit=$(GIT_COMMIT) \
 -X ${MODULE}/version.buildAt=$(BUILD_AT)

run: local
	./bundles/$(APP) -c ./internal/config/example.yaml

local:
	$(GO) build -ldflags="$(LD_FLAGS)" -v -o bundles/$(APP) .

build: local

test:
	$(GO) test -v ./...

build-image:
	docker build --force-rm -f ./Dockerfile -t ${IMAGE_TEST} .

push-image: build-image
	docker push ${IMAGE_TEST}

release-image: push-image
	docker tag ${IMAGE_TEST} ${IMAGE_RELEASE}
	docker push ${IMAGE_RELEASE}

release: clean
	GOOS=linux make build
	cd ./bundles && tar zcvf ./${APP}.linux-${shell go env GOARCH}.tar.gz ./${APP}
	GOOS=windows make build
	cd ./bundles && tar zcvf ./${APP}.windows-${shell go env GOARCH}.tar.gz ./${APP}
	GOOS=darwin make build
	cd ./bundles && tar zcvf ./${APP}.darwin-${shell go env GOARCH}.tar.gz ./${APP}

clean:
	rm -rf ./bundles

ver:
	@echo "Version:   " $(VERSION)
	@echo "Git commit:" $(GIT_COMMIT)
	@echo "Go version:" $(GO_VERSION)
	@echo "OS env:" $(shell go env GOOS)-$(shell go env GOARCH)
	@echo "Build time:" $(BUILD_TIME)
