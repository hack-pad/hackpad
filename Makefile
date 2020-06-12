SHELL := /usr/bin/env bash
GO_VERSION = 1.14.4
GOBIN = ${PWD}/cache/go${GO_VERSION}/bin
PATH := ${GOBIN}:${PATH}
GOOS = js
GOARCH = wasm
export
LINT_VERSION=1.27.0

.PHONY: serve
serve:
	go run ./server

.PHONY: lint
lint:
	@if ! which golangci-lint >/dev/null || [[ "$$(golangci-lint version 2>&1)" != *${LINT_VERSION}* ]]; then \
		curl -sfL https://raw.githubusercontent.com/golangci/golangci-lint/master/install.sh | sh -s -- -b $$(go env GOPATH)/bin v${LINT_VERSION}; \
	fi
	golangci-lint run

.PHONY: static
static: out/index.html out/go.zip commands

out:
	mkdir -p out

out/index.html: out ./server/index.html ./server/reload.js
	cp server/index.html ./server/reload.js ./out/

out/go.zip: out go
	cd cache/go${GO_VERSION}; \
		zip -ru -9 ../../out/go . -x \
			'.git/*' \
			'bin/*' \
			'pkg/*' \
			'src/cmd/*' \
			'test/*' \
			|| true

.PHONY: clean
clean:
	rm -rf out

cache:
	mkdir -p cache

.PHONY: commands
commands: out/go.wasm out/main.wasm $(patsubst cmd/%,out/%.wasm,$(wildcard cmd/*))

.PHONY: go
go: cache/go${GO_VERSION}

cache/go${GO_VERSION}: cache
	if [[ ! -e cache/go${GO_VERSION} ]]; then \
		set -ex; \
		TMP=$$(mktemp -d); trap 'rm -rf "$$TMP"' EXIT; \
		git clone \
			--depth 1 \
			--single-branch \
			--branch go${GO_VERSION} \
			git@github.com:golang/go.git \
			"$$TMP"; \
		pushd "$$TMP/src"; \
		mkdir -p ../bin/js_wasm; \
		./make.bash; \
		go build -o ../bin/js_wasm/ std cmd/go; \
		popd; \
		mv "$$TMP" cache/go${GO_VERSION}; \
	fi
	touch cache/go${GO_VERSION}

out/%.wasm: out go
	go build -o $@ ./cmd/$*

out/main.wasm: out go
	go build -o out/main.wasm .

out/go.wasm: out go
	cp cache/go${GO_VERSION}/bin/js_wasm/go out/go.wasm
	cp cache/go${GO_VERSION}/misc/wasm/wasm_exec.js out/wasm_exec.js

