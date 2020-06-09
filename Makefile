GO_VERSION = 1.14.4
GO = ${PWD}/cache/go${GO_VERSION}/bin/go
WASM_GO = GOOS=js GOARCH=wasm ${GO}

.PHONY: serve
serve:
	go run ./server

.PHONY: static
static: out/index.html commands

out:
	mkdir -p out

out/index.html: out ./index.html
	cp index.html ./out/

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
		GOOS=js GOARCH=wasm ../bin/go build -o ../bin/js_wasm/ std cmd/go; \
		popd; \
		mv "$$TMP" cache/go${GO_VERSION}; \
	fi
	touch cache/go${GO_VERSION}

out/%.wasm: out go
	$(WASM_GO) build -o $@ ./cmd/$*

out/main.wasm: out go
	$(WASM_GO) build -o out/main.wasm .

out/go.wasm: out go
	cp cache/go${GO_VERSION}/bin/js_wasm/go out/go.wasm
	cp cache/go${GO_VERSION}/misc/wasm/wasm_exec.js out/wasm_exec.js

