PKG?=$(shell go list ./... | grep -v /vendor/)
PROJECT := github.com/brettbuddin/eolian
BINPATH := bin

default: build

$(GOPATH)/bin/govendor:
	go get github.com/kardianos/govendor
govendor: $(GOPATH)/bin/govendor
	$(GOPATH)/bin/govendor sync

build: govendor
	@mkdir -p $(BINPATH)
	go build -o $(BINPATH)/eolian -v $(PROJECT)/cmd/eolian

test: govendor
	go test -test.timeout=1000s -cover $(PKG)

benchmark: govendor
	go test -test.timeout=1000s -bench=. $(PKG)

install: govendor
	go install $(INSTALL_FLAGS) -v $(PKG)

clean:
	go clean $(PROJECT)/...

.PHONY: build test install clean benchmark
