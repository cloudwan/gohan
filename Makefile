NO_COLOR=\033[0m
OK_COLOR=\033[32;01m
ERROR_COLOR=\033[31;01m
WARN_COLOR=\033[33;01m

all: gen lint build test

deps:
	@echo -e "$(OK_COLOR)==> Installing dependencies$(NO_COLOR)"
	./tools/dev_setup.sh

format:
	@echo -e "$(OK_COLOR)==> Formatting$(NO_COLOR)"
	govendor fmt +local
gen:
	@echo -e "$(OK_COLOR)==> Generating files$(NO_COLOR)"
	go-bindata -nometadata -pkg util -o util/bindata.go \
	etc/schema/... \
	etc/templates/... \
	public/...
	go fmt util/bindata.go

test: build
	@echo -e "$(OK_COLOR)==> Testing$(NO_COLOR)"

lint:
	@echo -e "$(OK_COLOR)==> Linting$(NO_COLOR)"
	./tools/lint.sh

build: deps
	@echo -e "$(OK_COLOR)==> Building$(NO_COLOR)"

install:
	@echo -e "$(OK_COLOR)==> Installing$(NO_COLOR)"
	./tools/install.sh
