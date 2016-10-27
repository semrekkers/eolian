PROJECT := github.com/brettbuddin/eolian
BINPATH := bin/

default: build

$(GOPATH)/bin/govendor:
	go get github.com/kardianos/govendor
govendor: $(GOPATH)/bin/govendor
	$(GOPATH)/bin/govendor sync

build: govendor
	@mkdir -p $(BINPATH)
	go build -o $(BINPATH)/eolian -v $(PROJECT)/cmd/eolian

test: govendor
	go test -test.timeout=1000s $(PROJECT)/...

install: govendor
	go install $(INSTALL_FLAGS) -v $(PROJECT)/...

clean:
	go clean $(PROJECT)/...

.PHONY: build test install clean
