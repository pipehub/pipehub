PROJECT_PATH      = /opt/httpway
DOCKER_CI_IMAGE   = registry.gitlab.com/httpway/httpway/ci
DOCKER_CI_VERSION = 1

configure:
	@git config pull.rebase true
	@git config remote.origin.prune true
	@git config branch.master.mergeoptions "--ff-only"

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
	@golangci-lint run -c .golangci.toml
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