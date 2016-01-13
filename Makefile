NO_COLOR=\033[0m
OK_COLOR=\033[32;01m
ERROR_COLOR=\033[31;01m
WARN_COLOR=\033[33;01m

# You need v8worker https://github.com/ry/v8worker installed in your env
# in order to enable v8 support on extension. then set ENABLE_V8=true
ifeq "$(ENABLE_V8)" "true"
	GO_BUILD=go build -tags=v8 ./...
else
	GO_BUILD=go build ./...
endif


all: format lint build test

deps:
	@echo "$(OK_COLOR)==> Installing dependencies$(NO_COLOR)"
	godep restore

savedeps:
	@echo "$(OK_COLOR)==> Updating all dependencies$(NO_COLOR)"
	godep save ./...

format:
	@echo "$(OK_COLOR)==> Formatting$(NO_COLOR)"
	go-bindata -pkg util -o util/go-bindata.go etc/schema/... etc/extensions/... etc/templates/...
	go fmt ./...

test:
	@echo "$(OK_COLOR)==> Testing$(NO_COLOR)"
	ENABLE_V8="$(ENABLE_V8)" ./run_test.sh

lint:
	@echo "$(OK_COLOR)==> Linting$(NO_COLOR)"
	golint ./... | grep -v util/go-bindata.go | grep -v extension/gohanscript/op.go | test `wc -l` -eq 0
	go vet ./...

build: deps
	@echo "$(OK_COLOR)==> Building$(NO_COLOR)"
	go run ./extension/gohanscript/tools/gen.go genlib -t extension/gohanscript/templates/lib.tmpl -p github.com/cloudwan/gohan/extension/gohanscript/lib -e autogen -ep extension/gohanscript/autogen
	$(GO_BUILD)

install:
	@echo "$(OK_COLOR)==> Installing$(NO_COLOR)"
	go install ./...
