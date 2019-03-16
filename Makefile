PROJECT_PATH       = /opt/pipehub
DOCKER_CI_IMAGE    = registry.gitlab.com/pipehub/pipehub/ci
DOCKER_CI_VERSION  = 1
CONFIG_PATH       ?= $(CURDIR)/cmd/pipehub/pipehub.hcl
WORKSPACE_PATH     = $(CURDIR)
RAWTAG             = $(shell git tag --points-at | head -n1 | cut -c2-)

configure:
	@git config pull.rebase true
	@git config remote.origin.prune true
	@git config branch.master.mergeoptions "--ff-only"

release:
	@PIPEHUB_DOCKER_IMAGE_VERSION=$(RAWTAG) goreleaser release --rm-dist

build:
	@go build -tags "$(TAGS)" -o cmd/pipehub/pipehub cmd/pipehub/*.go

generate:
	@rm -f pipe_dynamic.go
	@GOOS="" GOARCH="" make build
	@./cmd/pipehub/pipehub generate -c $(CONFIG_PATH) -w $(WORKSPACE_PATH)
	@GOOS=${GOOS} GOARCH=${GOARCH} TAGS=pipe make build

pre-pr: go-test go-linter go-linter-vendor docker-linter

go-test:
ifeq ($(EXEC_CONTAINER), false)
	@gotest -mod readonly -failfast -race -coverprofile=test.cover ./...
	@go tool cover -func=test.cover
	@rm -f test.cover
else
	TARGET=go-test make docker-exec
endif

go-linter:
ifeq ($(EXEC_CONTAINER), false)
	@golangci-lint run -c misc/golangci/golangci.toml
else
	TARGET=go-linter make docker-exec
endif

go-linter-vendor:
ifeq ($(EXEC_CONTAINER), false)
	@go mod tidy
	@go mod vendor
	@git diff --exit-code
else
	TARGET=go-linter-vendor make docker-exec
endif

docker-linter:
ifeq ($(EXEC_CONTAINER), false)
	@hadolint misc/docker/ci/Dockerfile
else
	TARGET=docker-linter make docker-exec
endif

docker-exec:
	@docker run \
		-t \
		--rm \
		-e EXEC_CONTAINER=false \
		-e TAGS=$(TAGS) \
		-e "TERM=xterm-256color" \
		-v $(PWD):$(PROJECT_PATH) \
		-w $(PROJECT_PATH) \
		$(DOCKER_CI_IMAGE):$(DOCKER_CI_VERSION) \
		make $(TARGET)

docker-ci-image:
	@docker build -t $(DOCKER_CI_IMAGE):$(DOCKER_CI_VERSION) -f misc/docker/ci/Dockerfile .
ifeq ($(PUSH), true)
	@docker push $(DOCKER_CI_IMAGE):$(DOCKER_CI_VERSION)
endif