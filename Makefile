# Common commands of the godart repository. Run "make" for the list.

RUNTIME := runtime/goruntime
GO_FILES := $(shell find . -name '*.go' -not -path './build/*')

.PHONY: help get build test vet fmt fmt-check dart-get dart-analyze dart-test \
	dart-fmt dart-fmt-check ci install clean

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

install:
	go install ./cmd/gart

clean:
	rm -rf build
