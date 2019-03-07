PROJECT_PATH       = /opt/httpway
DOCKER_CI_IMAGE    = registry.gitlab.com/httpway/httpway/ci
DOCKER_CI_VERSION  = 1
CONFIG_PATH       ?= $(CURDIR)/cmd/httpway/httpway.hcl
WORKSPACE_PATH     = $(CURDIR)

configure:
	@git config pull.rebase true
	@git config remote.origin.prune true
	@git config branch.master.mergeoptions "--ff-only"

build:
	@go build -tags "$(TAGS)" -o cmd/httpway/httpway cmd/httpway/*.go

generate:
	@rm -f handler_dynamic.go
	@$(MAKE) build
	@./cmd/httpway/httpway generate -c $(CONFIG_PATH) -w $(WORKSPACE_PATH)
	TAGS=handler @$(MAKE) build

pre-pr: go-test go-linter go-linter-vendor docker-linter

go-test:
ifeq ($(EXEC_CONTAINER), false)
	@gotest -mod readonly -failfast -race -coverprofile=test.cover ./...
	@go tool cover -func=test.cover
	@rm -f test.cover
else
	TARGET=go-test $(MAKE) docker-exec
endif

go-linter:
ifeq ($(EXEC_CONTAINER), false)
	@golangci-lint run -c misc/golangci/golangci.toml
else
	TARGET=go-linter $(MAKE) docker-exec
endif

go-linter-vendor:
ifeq ($(EXEC_CONTAINER), false)
	@go mod tidy
	@go mod vendor
	@git diff --exit-code
else
	TARGET=go-linter-vendor $(MAKE) docker-exec
endif

docker-linter:
ifeq ($(EXEC_CONTAINER), false)
	@hadolint misc/docker/ci/Dockerfile
else
	TARGET=docker-linter $(MAKE) docker-exec
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