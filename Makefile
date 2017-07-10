NO_COLOR=\033[0m
OK_COLOR=\033[32;01m
ERROR_COLOR=\033[31;01m
WARN_COLOR=\033[33;01m

all: gen lint build test

deps:
	@echo "$(OK_COLOR)==> Installing dependencies$(NO_COLOR)"
	./tools/dev_setup.sh

format:
	@echo "$(OK_COLOR)==> Formatting$(NO_COLOR)"

	govendor fmt +local

gen:
	@echo "$(OK_COLOR)==> Generating files$(NO_COLOR)"
	go-bindata -pkg util -o util/bindata.go \
	etc/schema/... \
	etc/extensions/... \
	etc/templates/... \
	public/...

test: build
	@echo "$(OK_COLOR)==> Testing$(NO_COLOR)"
	./run_test.sh

lint:
	@echo "$(OK_COLOR)==> Linting$(NO_COLOR)"
	./tools/lint.sh

build: deps
	@echo "$(OK_COLOR)==> Building$(NO_COLOR)"
	./tools/build.sh
	./tools/build_go_tests.sh

install:
	@echo "$(OK_COLOR)==> Installing$(NO_COLOR)"
	./tools/install.sh
