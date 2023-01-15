NAME       = edge-agent
BUILD_PATH = bin/$(NAME)
# NOTE: CGO needs to be disabled for running images on Alpine
GOENV      = GOARCH=amd64 GOOS=linux CGO_ENABLED=0
GOCMD      = go
GOBUILD    = $(GOCMD) build -o

DOCKER_REPOSITORY = warrantdev
DOCKER_IMAGE      = edge-agent
DOCKER_TAG        = $(VERSION)
VERSION           = $(shell cat VERSION)

.PHONY: clean
clean:
	rm -f $(BUILD_PATH)

.PHONY: dev
dev: clean
	$(GOCMD) get
	$(GOBUILD) $(BUILD_PATH) main.go

.PHONY: build
build: clean
	$(GOCMD) get
	$(GOENV) $(GOBUILD) $(BUILD_PATH) -ldflags="-s -w" main.go

.PHONY: docker
docker:
	docker build --platform linux/amd64 -t $(DOCKER_REPOSITORY)/$(DOCKER_IMAGE):$(DOCKER_TAG) .
	docker build --platform linux/amd64 -t $(DOCKER_REPOSITORY)/$(DOCKER_IMAGE) .

.PHONY: push
push: docker
	docker push $(DOCKER_REPOSITORY)/$(DOCKER_IMAGE):$(DOCKER_TAG)
	docker push $(DOCKER_REPOSITORY)/$(DOCKER_IMAGE)
