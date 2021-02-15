VERSION ?= "master"

ifndef GOPATH
	GOPATH := $(HOME)/go
endif

ifndef GO111MODULE
	GO111MODULE := on
endif

ifndef DOCKER_REGISTRY
	DOCKER_REGISTRY := hub.docker.com:5000
endif

all: build

build: build_linux_amd64

build_linux_amd64:
	@GOPATH=$(GOPATH) GOOS=linux GOARCH=amd64 CGO_ENABLED=1 go build -o $(GOPATH)/bin/dynamore-feature-extraction-runner.linux_amd64

github-release: build
	scripts/gh-release.sh $(VERSION) false

docker-release:
	docker build --force-rm -t $(DOCKER_REGISTRY)/dfe_runnerd:$(VERSION) . && \
		docker login $(DOCKER_REGISTRY) && \
		docker push $(DOCKER_REGISTRY)/dfe_runnerd:$(VERSION)

test:
	@GOPATH=$(GOPATH) GOOS=$(GOOS) GOARCH=amd64 go test -v github.com/Donders-Institute/dynamore-feature-extraction-runner/...

clean:
	@GOPATH=$(GOPATH) rm -f $(GOPATH)/bin/dynamore-feature-extraction-runner.linux_amd64
