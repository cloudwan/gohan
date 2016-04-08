NO_COLOR=\033[0m
OK_COLOR=\033[32;01m
ERROR_COLOR=\033[31;01m
WARN_COLOR=\033[33;01m

all: format lint build test

deps:
	@echo "$(OK_COLOR)==> Installing dependencies$(NO_COLOR)"
	./tools/dev_setup.sh

format:
	@echo "$(OK_COLOR)==> Formatting$(NO_COLOR)"
	go-bindata -pkg util -o util/go-bindata.go etc/schema/... etc/extensions/... etc/templates/... public/...
	go fmt ./...

test:
	@echo "$(OK_COLOR)==> Testing$(NO_COLOR)"
	./run_test.sh

lint:
	@echo "$(OK_COLOR)==> Linting$(NO_COLOR)"
	./tools/lint.sh

build: deps
	@echo "$(OK_COLOR)==> Building$(NO_COLOR)"
	./tools/build.sh

install:
	@echo "$(OK_COLOR)==> Installing$(NO_COLOR)"
	./tools/install.sh
