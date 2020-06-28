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

.PHONY: lint-deps
lint-deps:
	@if ! which golangci-lint >/dev/null || [[ "$$(golangci-lint version 2>&1)" != *${LINT_VERSION}* ]]; then \
		curl -sfL https://raw.githubusercontent.com/golangci/golangci-lint/master/install.sh | sh -s -- -b $$(go env GOPATH)/bin v${LINT_VERSION}; \
	fi

.PHONY: lint
lint: lint-deps
	golangci-lint run

.PHONY: lint-fix
lint-fix: lint-deps
	golangci-lint run --fix

.PHONY: static
static: out/index.html out/go.zip commands

out:
	mkdir -p out

out/index.html: out ./server/index.html ./server/js/*.js ./out/fetch.js
	cp -r server/index.html ./server/js ./out/

out/fetch.js:
	curl -L https://github.com/github/fetch/releases/download/v3.0.0/fetch.umd.js > out/fetch.js

out/go.zip: out go
	cd cache; \
		zip -ru -9 ../out/go ./go -x \
			'./go/.git/*' \
			'./go/bin/*' \
			'./go/pkg/*' \
			'./go/src/cmd/*' \
			'./go/test/*'; \
		zip -ru -9 ../out/go \
			./go/bin/js_wasm \
			./go/pkg/js_wasm \
			./go/pkg/include \
			./go/pkg/tool/js_wasm \
			; \
		true

.PHONY: clean
clean:
	rm -rf out

cache:
	mkdir -p cache

.PHONY: commands
commands: out/wasm_exec.js out/main.wasm $(patsubst cmd/%,out/%.wasm,$(wildcard cmd/*))

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
		$(MAKE) -e TMP_GO="$$TMP" go-ext; \
		pushd "$$TMP/src"; \
		./make.bash; \
		export PATH="$$TMP/bin:$$PATH"; \
		go version; \
		mkdir -p ../bin/js_wasm; \
		go build -o ../bin/js_wasm/ std cmd/go; \
		go tool dist test -rebuild -list; \
		go build -o ../pkg/tool/js_wasm/ std cmd/buildid cmd/pack; \
		popd; \
		mv "$$TMP" cache/go${GO_VERSION}; \
		ln -sf go${GO_VERSION} go; \
	fi
	touch cache/go${GO_VERSION}

out/%.wasm: out go
	go build -o $@ ./cmd/$*

out/main.wasm: out go
	go build -o out/main.wasm .

out/wasm_exec.js: out go
	cp cache/go/misc/wasm/wasm_exec.js out/wasm_exec.js

.PHONY: go-ext
go-ext:
	[[ -d "${TMP_GO}" ]]
	sed -i '' -e '/^func Pipe(/,/^}/d' "${TMP_GO}"/src/syscall/fs_js.go
	sed -i '' -e '/^func StartProcess(/,/^}/d' "${TMP_GO}"/src/syscall/syscall_js.go
	sed -i '' -e '/^func Wait4(/,/^}/d' "${TMP_GO}"/src/syscall/syscall_js.go
	cp internal/testdata/fs_* "${TMP_GO}"/src/syscall/
	cp internal/testdata/syscall_* "${TMP_GO}"/src/syscall/
	cp internal/testdata/filelock_* "${TMP_GO}"/src/cmd/go/internal/lockedfile/internal/filelock/
	sed -i '' -e 's/+build\( [^j].*\)*$$/+build\1 js,wasm/' "${TMP_GO}"/src/os/exec/lp_unix.go
	rm "${TMP_GO}"/src/os/exec/lp_js.go
