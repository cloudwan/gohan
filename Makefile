prefix=${prefix, "/usr/local"}
NO_COLOR=\033[0m
OK_COLOR=\033[32;01m
ERROR_COLOR=\033[31;01m
WARN_COLOR=\033[33;01m

# You need v8worker https://github.com/ry/v8worker installed in your env
# in order to enable v8 support on extension. then set ENABLE_V8=true
ifeq "$(ENABLE_V8)" "true"
	GO_BUILD=gb build -tags 'v8'
else
	GO_BUILD=gb build
endif


all: format lint build test

deps:
	@echo "$(OK_COLOR)==> Installing dependencies$(NO_COLOR)"
	gb vendor restore

format:
	@echo "$(OK_COLOR)==> Formatting$(NO_COLOR)"
	go fmt ./src/...

test:
	@echo "$(OK_COLOR)==> Testing$(NO_COLOR)"
	ENABLE_V8="$(ENABLE_V8)" ./run_test.sh

lint:
	@echo "$(OK_COLOR)==> Linting$(NO_COLOR)"
	golint ./src/...
	go vet ./src/...

build:
	@echo "$(OK_COLOR)==> Building$(NO_COLOR)"
	gb build

install:
	@echo "$(OK_COLOR)==> Installing$(NO_COLOR)"
	install bin/gohan $(prefix)/bin
