# Common commands of the godart repository. Run "make" for the list.

RUNTIME := runtime/goruntime
GO_FILES := $(shell find . -name '*.go' -not -path './build/*')

PROG ?= hello
# DEMO holds the output of run and of compare. The identifier of the process
# ends the path, thus 2 runs at the same time write no mixed tree.
DEMO := $(CURDIR)/build/demo/$(PROG)-$(shell echo $$$$)

SUITE ?= basic
SUITE_OUT := $(CURDIR)/build/suite/$(SUITE)-$(shell echo $$$$)

STD ?= path
STD_OUT := $(CURDIR)/build/std/$(subst /,-,$(STD))-$(shell echo $$$$)

FLUTTER ?= flutter

.PHONY: help get build test vet fmt fmt-check dart-get dart-analyze dart-test \
	dart-fmt dart-fmt-check ci install clean run compare programs suites \
	gart-test compare-tests bench std-test flutter-go flutter-test flutter-nodes \
	surface

help:
	@echo "godart targets:"
	@echo "  make get              Download the Go and the Dart dependencies."
	@echo "  make build            Compile the gart command to build/gart."
	@echo "  make test             Run the Go tests."
	@echo "  make vet              Run go vet."
	@echo "  make fmt              Format the Go code and the Dart code."
	@echo "  make fmt-check        Fail if the format differs."
	@echo "  make dart-analyze     Run the Dart analyzer on the runtime library."
	@echo "  make dart-test        Run the Dart tests of the runtime library."
	@echo "  make flutter-go       Run the Go tests of the Flutter modules."
	@echo "  make flutter-test     Run the core Flutter validation."
	@echo "  make flutter-nodes    Write the Go node set again from the Flutter API."
	@echo "  make surface          Measure the standard library surface."
	@echo "  make surface ARGS=-absent   Name each package that the surface holds not."
	@echo "  make ci               Run the full check that CI runs."
	@echo "  make programs         List the Go programs of testdata."
	@echo "  make run PROG=x       Compile testdata/programs/x and run the Dart output."
	@echo "  make compare PROG=x   Run the program with go and with Dart, and diff."
	@echo "  make suites           List the test modules of testdata."
	@echo "  make gart-test SUITE=x        Run the tests of the module with gart."
	@echo "  make compare-tests SUITE=x    Run go test and gart test, and diff."
	@echo "  make bench SUITE=x            Run the benchmarks of the module with gart."
	@echo "  make std-test STD=x           Run the real tests of a GOROOT package with gart."
	@echo "  make install          Put the gart and glutter commands on the PATH."
	@echo "  make clean            Delete the build directory."

get: dart-get
	go mod download

dart-get:
	cd $(RUNTIME) && dart pub get

build:
	go build -o build/gart ./cmd/gart
	go build -o build/glutter ./cmd/glutter

# Full upstream suites exceed the default Go test timeout.
test:
	go test -v -p=1 -count=1 -timeout 90m ./...

vet:
	go vet ./...

fmt: dart-fmt
	gofmt -w $(GO_FILES)

dart-fmt:
	dart format internal/driver/resident.dart
	cd $(RUNTIME) && dart format lib test

fmt-check: dart-fmt-check
	@out=$$(gofmt -l $(GO_FILES)); \
	if [ -n "$$out" ]; then echo "gofmt needs these files:"; echo "$$out"; exit 1; fi

dart-fmt-check:
	dart format --output=none --set-exit-if-changed internal/driver/resident.dart
	cd $(RUNTIME) && dart format --output=none --set-exit-if-changed lib test

dart-analyze:
	dart analyze --fatal-infos internal/driver/resident.dart
	cd $(RUNTIME) && dart analyze --fatal-infos

dart-test:
	cd $(RUNTIME) && dart test

# flutter-go runs the Go side of the Flutter work. Each one is a module of its
# own, thus the go targets of the root do not reach it. It needs no Flutter SDK.
flutter-go:
	cd flutter/ui && go vet ./... && go test -count=1 ./...
	cd flutter/testdata/core/go && go vet ./... && go test -count=1 ./...

# This target needs the Flutter SDK. CI runs the Go validation separately.
flutter-test: build
	go test ./cmd/glutter -run '^TestNewApp$$' -count=1
	cd flutter/goflutter && $(FLUTTER) pub get
	cd flutter/goflutter && $(FLUTTER) analyze
	cd flutter/bench/flutter && $(FLUTTER) pub get && $(FLUTTER) analyze
	cd flutter/nodegen && dart pub get && dart analyze
	cd flutter/nodegen && dart run test/notices_test.dart
	cd flutter/nodegen && dart run test/capabilities_test.dart
	cd flutter/goflutter && $(FLUTTER) test
	cd flutter/bench/flutter && $(FLUTTER) test
	cd flutter/testdata/core && ../../../build/glutter gen
	cd flutter/testdata/core && ../../../build/glutter bind
	cd flutter/nodegen && dart run test/bindings_test.dart
	cd flutter/testdata/core && $(FLUTTER) pub get && $(FLUTTER) test

# surface measures how much of each package of the standard library the
# compiler holds.
surface:
	go run ./internal/surface $(ARGS)

# flutter-nodes writes the Go node set again from the public API of Flutter.
# Take it after an upgrade of the Flutter SDK. Read flutter/nodegen/README.md.
flutter-nodes:
	cd flutter/nodegen && dart pub get && dart run bin/nodegen.dart go \
		../goflutter ../ui

ci:
	@for suite in checks driver integration targets web-io web-fetch programs language stdlib upstream-priority upstream-targets upstream-blockers; do \
		bash scripts/ci.sh $$suite || exit $$?; \
	done

programs:
	@ls testdata/programs

# run compiles 1 program of testdata and runs the Dart output.
run: build
	@rm -rf $(DEMO)
	@cd testdata/programs/$(PROG) && \
		$(CURDIR)/build/gart build -o $(DEMO) -runtime $(CURDIR)/$(RUNTIME) ./...
	@cd $(DEMO) && dart pub get > /dev/null && dart run bin/main.dart

# compare runs the program with the Go toolchain and with Dart, and it reports
# each difference.
compare: build
	@rm -rf $(DEMO)
	@cd testdata/programs/$(PROG) && go run . > $(DEMO).go.txt 2>&1 || true
	@cd testdata/programs/$(PROG) && \
		$(CURDIR)/build/gart build -o $(DEMO) -runtime $(CURDIR)/$(RUNTIME) ./...
	@cd $(DEMO) && dart pub get > /dev/null && \
		dart run bin/main.dart > $(DEMO).dart.txt 2>&1 || true
	@diff $(DEMO).go.txt $(DEMO).dart.txt && echo "$(PROG): the 2 outputs are equal"

suites:
	@ls testdata/tests

# gart-test compiles the tests of 1 module of testdata and runs them with the
# Dart VM.
gart-test: build
	@cd testdata/tests/$(SUITE) && \
		$(CURDIR)/build/gart test -o $(SUITE_OUT) -runtime $(CURDIR)/$(RUNTIME) -v ./...

# NORMALIZE cuts each part of the output that a run does not repeat. It holds
# the rule of internal/driver/test_test.go: the source position of a log line
# stays, and a position that names a file outside the test files goes. Go writes
# the position of its own runtime for a deferred log line after Fatal.
NORMALIZE = sed -E \
	-e 's/\([0-9]+\.[0-9][0-9]s\)/(0.00s)/' \
	-e 's/\t[0-9]+\.[0-9]+s/\t0.000s/' \
	-e '/^[[:space:]]+[A-Za-z0-9_.\/-]+_test\.go:[0-9]+: /!s/^([[:space:]]+)[A-Za-z0-9_.\/-]+\.go:[0-9]+: /\1/'

# compare-tests runs the tests with the go toolchain and with gart, and it
# reports each difference.
compare-tests: build
	@mkdir -p $(dir $(SUITE_OUT))
	@cd testdata/tests/$(SUITE) && go test -count=1 -v ./... > $(SUITE_OUT).go.txt 2>&1
	@cd testdata/tests/$(SUITE) && \
		$(CURDIR)/build/gart test -o $(SUITE_OUT) -runtime $(CURDIR)/$(RUNTIME) -v ./... \
		> $(SUITE_OUT).dart.txt 2>&1
	@$(NORMALIZE) $(SUITE_OUT).go.txt > $(SUITE_OUT).go.norm.txt
	@$(NORMALIZE) $(SUITE_OUT).dart.txt > $(SUITE_OUT).dart.norm.txt
	@diff $(SUITE_OUT).go.norm.txt $(SUITE_OUT).dart.norm.txt && \
		echo "$(SUITE): the 2 outputs are equal"

# std-test runs the real test files of 1 package of GOROOT with gart. Such a
# package belongs to no module, and the driver gives it the module std.
std-test: build
	@cd $(shell go env GOROOT)/src/$(STD) && \
		$(CURDIR)/build/gart test -o $(STD_OUT) -runtime $(CURDIR)/$(RUNTIME) -v .

# bench runs the benchmarks of 1 module with gart. It builds ahead of time
# first, because the warm up cost of the Dart VM hides the result.
bench: build
	@cd testdata/tests/$(SUITE) && \
		$(CURDIR)/build/gart test -o $(SUITE_OUT) -runtime $(CURDIR)/$(RUNTIME) \
		-bench . ./...

install:
	go install ./cmd/gart ./cmd/glutter

clean:
	rm -rf build
