PKG?=$(shell go list ./... | grep -v /vendor/)
PROJECT := github.com/brettbuddin/eolian
BINPATH := bin

default: build

$(GOPATH)/bin/govendor:
	go get github.com/kardianos/govendor
govendor: $(GOPATH)/bin/govendor
	$(GOPATH)/bin/govendor sync

$(GOPATH)/bin/go-bindata:
	go get github.com/jteeuwen/go-bindata/...
lua-scripts: $(GOPATH)/bin/go-bindata
	$(GOPATH)/bin/go-bindata -pkg lua -o lua/lib.go lua/lib/...

build: govendor lua-scripts
	@mkdir -p $(BINPATH)
	go build -o $(BINPATH)/eolian -v $(PROJECT)/cmd/eolian

test: govendor lua-scripts
	go test -test.timeout=1000s -cover $(PKG)

benchmark: govendor lua-scripts
	go test -test.timeout=1000s -bench=. $(PKG)

install: govendor lua-scripts
	go install $(INSTALL_FLAGS) -v $(PKG)

clean:
	go clean $(PROJECT)/...

.PHONY: build test install clean benchmark
