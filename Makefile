PKG?=$(shell go list ./... | grep -v /vendor/)
PROJECT := buddin.us/eolian
BINPATH := bin

default: test

$(GOPATH)/bin/go-bindata:
	go get github.com/jteeuwen/go-bindata/...
lua-scripts: $(GOPATH)/bin/go-bindata
	$(GOPATH)/bin/go-bindata -nometadata -pkg lua -o lua/lib.go lua/lib/...

build: lua-scripts
	@mkdir -p $(BINPATH)
	go build -o $(BINPATH)/eolian -v $(PROJECT)

test: lua-scripts
	go test -cover $(TESTARGS) $(PKG)

benchmark: lua-scripts
	go test -bench=. $(TESTARGS) $(PKG)

install: lua-scripts
	go install $(INSTALL_FLAGS) -v $(PKG)

clean:
	go clean $(PROJECT)/...

.PHONY: build test install clean benchmark
