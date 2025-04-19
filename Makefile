# Load environment variables from .env file
ifneq (,$(wildcard ../.env))
	include ../.env
	export $(shell sed 's/=.*//' ../.env)
endif

# Dynamic variables
APP_NAME := $(shell grep -lR "func main()" *.go | awk -F/ '{print $$NF}' | sed 's/\.go//')
PACKAGES := $(shell go list ./...)
NAME := $(shell basename ${PWD})
COMMIT_HASH := $(shell git rev-parse --short HEAD)
TIMESTAMP ?= $(shell date +"%Y%m%d%H%M%S")
VERSION ?= $(shell git describe --tags --always || git rev-parse --short HEAD)
LDFLAGS ?= -X 'main.Version=$(VERSION)'

# Docker parameters
AWS_REGION=us-east-1
AWS_ACCOUNT_ID=123456789012
EXECUTABLE={{.serviceName}}
IMAGE_NAME={{.serviceName}}
NAMESPACE={{.serviceName}}
DOCKER=docker
DOCKER_BUILD=$(DOCKER) build
AWS_ECR_REPO=${NAMESPACE}/${IMAGE_NAME}
AWS_ECR_TAG=latest
AWS_ECR_URL=$(AWS_ACCOUNT_ID).dkr.ecr.$(AWS_REGION).amazonaws.com/$(AWS_ECR_REPO)
AWS_LOGIN=$(shell aws ecr get-login-password --region $(AWS_REGION))
DSN ?= $(shell echo $(DSN) | sed 's|sqlite://||')

# Digital Ocean parameters
DO_REGION=sfo3
DO_REGISTRY=registry.digitalocean.com
DO_IMAGE=$(DO_REGISTRY)/$(NAMESPACE)/$(IMAGE_NAME)

# XO includes (parsed from .env)
XO_INCLUDES := $(shell echo "${XO_INCLUDES}" | xargs | sed -e 's/ /\ --include=/g')

all: help

.PHONY: help
help: Makefile
	@echo
	@echo " Application Name: $(APP_NAME)"
	@echo
	@echo " Choose a make command to run"
	@echo
	@sed -n 's/^##//p' $< | column -t -s ':' |  sed -e 's/^/ /'
	@echo

## vet: vet code
.PHONY: vet
vet:
	go vet $(PACKAGES)

## test: run unit tests
.PHONY: test
test:
	go test -race -cover $(PACKAGES)

## build: build project
.PHONY: build
build:
	go build -ldflags "-X main.Environment=production" -o ./tmp/$(APP_NAME) .

## staticcheck: run staticcheck
.PHONY: staticcheck
staticcheck:
	staticcheck ./...

.PHONY: docker-build
docker-build:
	$(DOCKER_BUILD) --platform=linux/amd64 -t $(AWS_ECR_URL):latest -t $(AWS_ECR_URL):main-$(TIMESTAMP)-$(COMMIT_HASH) .

.PHONY: docker-push
docker-push:
	@aws ecr get-login-password --region $(AWS_REGION) | docker login --username AWS --password-stdin $(AWS_ECR_URL)
	$(DOCKER) push $(AWS_ECR_URL):latest
	$(DOCKER) push $(AWS_ECR_URL):main-$(TIMESTAMP)-$(COMMIT_HASH)
	docker rmi $(AWS_ECR_URL):latest
	docker rmi $(AWS_ECR_URL):main-$(TIMESTAMP)-$(COMMIT_HASH)

.PHONY: do-docker-build
do-docker-build:
	$(DOCKER_BUILD) --platform=linux/amd64 -t $(DO_IMAGE):latest -t $(DO_IMAGE):main-$(TIMESTAMP)-$(COMMIT_HASH) .

.PHONY: do-docker-push
do-docker-push:
	$(DOCKER) push $(DO_IMAGE):latest
	$(DOCKER) push $(DO_IMAGE):main-$(TIMESTAMP)-$(COMMIT_HASH)
	docker rmi $(DO_IMAGE):latest
	docker rmi $(DO_IMAGE):main-$(TIMESTAMP)-$(COMMIT_HASH)
