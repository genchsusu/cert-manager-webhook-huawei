GO ?= $(shell which go)

# Docker 设置
IMAGE_NAME := genchsusu/cert-manager-webhook-huawei
IMAGE_TAG := 1.17.1

# Go 编译器设置
GOFLAGS = -ldflags="-s -w"
GOBUILD = $(GO) build $(GOFLAGS)
GORUN = $(GO) run $(GOFLAGS)

# 源码路径
SRC = ./cmd/

run:
	$(GORUN) $(SRC)

build:
	$(GOBUILD) -o ./bin/webhook $(SRC)

docker-build:
	docker build --platform linux/amd64 -t "$(IMAGE_NAME):$(IMAGE_TAG)" .

docker-push:
	docker push "$(IMAGE_NAME):$(IMAGE_TAG)"

image: docker-build docker-push

.PHONY: run build image docker-build docker-push