VERSION ?= "master"

ifndef GOPATH
	GOPATH := $(HOME)/go
endif

ifndef GO111MODULE
	GO111MODULE := on
endif

all: build

build: build_linux_amd64

build_linux_amd64:
	@GOPATH=$(GOPATH) GOOS=linux GOARCH=amd64 CGO_ENABLED=0 go build -o dynamore-feature-extraction-runner.linux_amd64

# github-release: build
# 	scripts/gh-release.sh $(VERSION) false

test:
	@GOPATH=$(GOPATH) GOOS=$(GOOS) GOARCH=amd64 go test -v github.com/Donders-Institute/dynamore-feature-extraction-runner/...

clean:
	rm -f dynamore-feature-extraction-runner.linux_amd64
