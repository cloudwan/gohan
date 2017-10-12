NO_COLOR=\033[0m
OK_COLOR=\033[32;01m
ERROR_COLOR=\033[31;01m
WARN_COLOR=\033[33;01m

# tests require a build with race flag enabled
TEST_BUILD_FLAGS=-race

# don't build in parallel - CircleCI offers limited memory
ifdef CIRCLECI
    TEST_BUILD_FLAGS+=-p 1
endif

all: gen lint build test

deps:
	@echo -e "$(OK_COLOR)==> Installing dependencies$(NO_COLOR)"
	./tools/dev_setup.sh

format:
	@echo -e "$(OK_COLOR)==> Formatting$(NO_COLOR)"
	govendor fmt +local
gen:
	@echo -e "$(OK_COLOR)==> Generating files$(NO_COLOR)"
	go-bindata -pkg util -o util/bindata.go \
	etc/schema/... \
	etc/extensions/... \
	etc/templates/... \
	public/...

test: deps
	@echo -e "$(OK_COLOR)==> Testing$(NO_COLOR)"
	./tools/build.sh $(TEST_BUILD_FLAGS)
	./tools/build_go_tests.sh "$(TEST_BUILD_FLAGS)"
	./tools/test_bash_completion.sh
	./run_test.sh

lint:
	@echo -e "$(OK_COLOR)==> Linting$(NO_COLOR)"
	./tools/lint.sh

build: deps
	@echo -e "$(OK_COLOR)==> Building$(NO_COLOR)"
	./tools/build.sh
	./tools/build_go_tests.sh

install:
	@echo -e "$(OK_COLOR)==> Installing$(NO_COLOR)"
	./tools/install.sh
