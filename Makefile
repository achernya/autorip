# Set make's shell to bash, and put bash into pedantic mode.
SHELL = /bin/bash
.SHELLFLAGS = -euf -o pipefail -c

# Disable all built-in implicit rules.
.SUFFIXES:

# Allow the user to override the path to go.
GO ?= go

# Output program name
TARGET := autorip
# Get a list of sources from go. Using this over wildcard allows for
# the test files to not be included in the listing, meaning make won't
# try to rebuild the binary after test-only changes.
#
# This can be a simply-expanded variable assignment even though the
# generated sources may not yet exist because generated sources will
# be depended-upon explicitly. In an ideal world, the generated
# sources would be implicitly here, but this results in a
# bootstrapping problem for clean builds.
SOURCES := $(shell $(GO) list -f '{{$$d := .Dir}}{{range $$f := .GoFiles}}{{printf "%s/%s\n" $$d $$f}}{{end}}' ./...)
TEST_SOURCES := $(shell $(GO) list -f '{{$$d := .Dir}}{{range $$f := .TestGoFiles}}{{printf "%s/%s\n" $$d $$f}}{{end}}' ./...)
# Currently, all generated files are protobufs.
protos := $(wildcard */*.proto)
# .proto files turn into .pb.go files
GENERATED_SOURCES := $(protos:.proto=.pb.go)
# Coverage stuff goes into the build/ directory to hide the clutter.
COVERAGE_PROFILE := build/cover.out
COVERAGE_REPORT := build/cover.html

%.pb.go: %.proto
	$(CMD_V_GO_GEN)$(GO) generate ./...

$(TARGET): $(SOURCES) $(GENERATED_SOURCES)
	$(CMD_V_GO_BLD)$(GO) build -o $@

# `go test` handles caching, so test can/should be marked phony.
.PHONY: test
test: $(GENERATED_SOURCES)
	$(CMD_V_GO_TEST)$(GO) test ./...

# check is an alias for test
.PHONY: check
check: test

# `go test` with coverage enabled does not handle caching, so the full
# dependency list should be specified here.
$(COVERAGE_PROFILE): $(SOURCES) $(GENERATED_SOURCES) $(TEST_SOURCES)
	@mkdir -p build/
	$(CMD_V_GO_TEST)$(GO) test ./... -coverprofile=$@

$(COVERAGE_REPORT): $(COVERAGE_PROFILE)
	$(CMD_V_GO_COV)$(GO) tool cover -html=$< -o $@

.PHONY: coverage
coverage: $(COVERAGE_REPORT)

.PHONY: clean
clean:
	-rm $(COVERAGE_PROFILE)
	-rm $(COVERAGE_REPORT)
	-rm $(GENERATED_SOURCES)
	-rm $(TARGET)

.PHONY: all
all: autorip

# Some fairly ugly variables for pretty-printing make output.
CMD_DEFAULT_VERBOSITY = 0
CMD_V_GO_BLD = $(cmd__v_go_bld_$(V))
cmd__v_go_bld_ = $(cmd__v_go_bld_$(CMD_DEFAULT_VERBOSITY))
cmd__v_go_bld_0 = @echo "  GO BLD  " $@;
cmd__v_go_bld_1 = 
CMD_V_GO_GEN = $(cmd__v_go_gen_$(V))
cmd__v_go_gen_ = $(cmd__v_go_gen_$(CMD_DEFAULT_VERBOSITY))
cmd__v_go_gen_0 = @echo "  GO GEN  " $@;
cmd__v_go_gen_1 = 
CMD_V_GO_TEST = $(cmd__v_go_test_$(V))
cmd__v_go_test_ = $(cmd__v_go_test_$(CMD_DEFAULT_VERBOSITY))
cmd__v_go_test_0 = @echo "  GO TEST " $@;
cmd__v_go_test_1 = 
CMD_V_GO_COV = $(cmd__v_go_cov_$(V))
cmd__v_go_cov_ = $(cmd__v_go_cov_$(CMD_DEFAULT_VERBOSITY))
cmd__v_go_cov_0 = @echo "  GO COV  " $@;
cmd__v_go_cov_1 = 
