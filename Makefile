# Common commands of the godart repository. Run "make" for the list.

RUNTIME := runtime/goruntime
GO_FILES := $(shell find . -name '*.go' -not -path './build/*')

PROG ?= hello
DEMO := $(CURDIR)/build/demo/$(PROG)

SUITE ?= basic
SUITE_OUT := $(CURDIR)/build/suite/$(SUITE)

.PHONY: help get build test vet fmt fmt-check dart-get dart-analyze dart-test \
	dart-fmt dart-fmt-check ci install clean run compare programs suites \
	gart-test compare-tests

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
	@echo "  make ci               Run the full check that CI runs."
	@echo "  make programs         List the Go programs of testdata."
	@echo "  make run PROG=x       Compile testdata/programs/x and run the Dart output."
	@echo "  make compare PROG=x   Run the program with go and with Dart, and diff."
	@echo "  make suites           List the test modules of testdata."
	@echo "  make gart-test SUITE=x        Run the tests of the module with gart."
	@echo "  make compare-tests SUITE=x    Run go test and gart test, and diff."
	@echo "  make install          Put the gart command on the PATH."
	@echo "  make clean            Delete the build directory."

get: dart-get
	go mod download

dart-get:
	cd $(RUNTIME) && dart pub get

build:
	go build -o build/gart ./cmd/gart

test:
	go test ./...

vet:
	go vet ./...

fmt: dart-fmt
	gofmt -w $(GO_FILES)

dart-fmt:
	cd $(RUNTIME) && dart format lib test

fmt-check: dart-fmt-check
	@out=$$(gofmt -l $(GO_FILES)); \
	if [ -n "$$out" ]; then echo "gofmt needs these files:"; echo "$$out"; exit 1; fi

dart-fmt-check:
	cd $(RUNTIME) && dart format --output=none --set-exit-if-changed lib test

dart-analyze:
	cd $(RUNTIME) && dart analyze --fatal-infos

dart-test:
	cd $(RUNTIME) && dart test

ci: fmt-check vet test dart-analyze dart-test

programs:
	@ls testdata/programs

# run compiles 1 program of testdata and runs the Dart output.
run: build
	@cd testdata/programs/$(PROG) && \
		$(CURDIR)/build/gart build -o $(DEMO) -runtime $(CURDIR)/$(RUNTIME) ./...
	@cd $(DEMO) && dart pub get > /dev/null && dart run bin/main.dart

# compare runs the program with the Go toolchain and with Dart, and it reports
# each difference.
compare: build
	@cd testdata/programs/$(PROG) && go run . > $(CURDIR)/build/go.txt 2>&1 || true
	@cd testdata/programs/$(PROG) && \
		$(CURDIR)/build/gart build -o $(DEMO) -runtime $(CURDIR)/$(RUNTIME) ./...
	@cd $(DEMO) && dart pub get > /dev/null && \
		dart run bin/main.dart > $(CURDIR)/build/dart.txt 2>&1 || true
	@diff $(CURDIR)/build/go.txt $(CURDIR)/build/dart.txt && echo "$(PROG): the 2 outputs are equal"

suites:
	@ls testdata/tests

# gart-test compiles the tests of 1 module of testdata and runs them with the
# Dart VM.
gart-test: build
	@cd testdata/tests/$(SUITE) && \
		$(CURDIR)/build/gart test -o $(SUITE_OUT) -runtime $(CURDIR)/$(RUNTIME) -v ./...

# compare-tests runs the tests with the go toolchain and with gart, and it
# reports each difference. A log line of go test holds a source position, and a
# log line of gart holds none, thus the diff cuts the position away.
compare-tests: build
	@cd testdata/tests/$(SUITE) && go test -count=1 -v ./... > $(CURDIR)/build/go.txt 2>&1 || true
	@cd testdata/tests/$(SUITE) && \
		$(CURDIR)/build/gart test -o $(SUITE_OUT) -runtime $(CURDIR)/$(RUNTIME) -v ./... \
		> $(CURDIR)/build/dart.txt 2>&1 || true
	@sed -E -e 's/\([0-9]+\.[0-9][0-9]s\)/(0.00s)/' -e 's/\t[0-9]+\.[0-9]+s/\t0.000s/' \
		-e 's/^([[:space:]]+)[A-Za-z0-9_./]+\.go:[0-9]+: /\1/' \
		$(CURDIR)/build/go.txt > $(CURDIR)/build/go.norm.txt
	@sed -E -e 's/\([0-9]+\.[0-9][0-9]s\)/(0.00s)/' -e 's/\t[0-9]+\.[0-9]+s/\t0.000s/' \
		$(CURDIR)/build/dart.txt > $(CURDIR)/build/dart.norm.txt
	@diff $(CURDIR)/build/go.norm.txt $(CURDIR)/build/dart.norm.txt && \
		echo "$(SUITE): the 2 outputs are equal"

install:
	go install ./cmd/gart

clean:
	rm -rf build
