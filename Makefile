GO ?= go
# v8go 的 C++ 头文件在 C++20 下有 -Wtemplate-id-cdtor 兼容性告警（仅编译期噪声）；
# 这里定向关闭，保留其它警告。第二个参数用于容忍 clang 不认识 GCC 告警名。
export CGO_CXXFLAGS ?= -Wno-template-id-cdtor -Wno-unknown-warning-option
GOFMT ?= gofmt "-s"
PACKAGES ?= $(shell $(GO) list ./...)
VETPACKAGES ?= $(shell $(GO) list ./... | grep -v /examples/)
GOFILES := $(shell find . -name "*.go")

# ROOT_DIR := $(shell dirname $(realpath $(firstword $(MAKEFILE_LIST))))
# Ignore |mcp|ffmpeg|office|pdf|graphrag add back later
TESTFOLDER := $(shell $(GO) list ./... | grep -E 'api|server/http|runtime|process|widget|model|plan|schema|lang|query|task|schedule|flow|session|store|fs|http|encoding|ssl|plugin|connector|wasm|websocket$|v8|application|diff' | grep -v -E 'wamr|socket')
TESTTAGS ?= ""

.PHONY: test deps
test:
	echo "mode: count" > coverage.out
	for d in $(TESTFOLDER); do \
		$(GO) test -tags $(TESTTAGS) -v -covermode=count -coverprofile=profile.out -coverpkg=$$(echo $$d | sed "s/\/test$$//g") $$d > tmp.out; \
		cat tmp.out; \
		if grep -q "^--- FAIL" tmp.out; then \
			rm tmp.out; \
			exit 1; \
		elif grep -q "^FAIL	" tmp.out; then \
			rm tmp.out; \
			exit 1; \
		elif grep -q "^panic:" tmp.out; then \
			rm tmp.out; \
			exit 1; \
		elif grep -q "build failed" tmp.out; then \
			rm tmp.out; \
			exit 1; \
		elif grep -q "setup failed" tmp.out; then \
			rm tmp.out; \
			exit 1; \
		elif grep -q "runtime error" tmp.out; then \
			rm tmp.out; \
			exit 1; \
		fi; \
		if [ -f profile.out ]; then \
			cat profile.out | grep -v "mode:" >> coverage.out; \
			rm profile.out; \
		fi; \
	done

.PHONY: bench
bench:
	for d in $(TESTFOLDER); do \
		$(GO) test -run Benchmark -v -bench=. -benchtime=5s -benchmem $$d; \
	done

.PHONY: fmt
fmt:
	$(GOFMT) -w $(GOFILES)

.PHONY: fmt-check
fmt-check:
	@diff=$$($(GOFMT) -d $(GOFILES)); \
	if [ -n "$$diff" ]; then \
		echo "Please run 'make fmt' and commit the result:"; \
		echo "$${diff}"; \
		exit 1; \
	fi;

vet:
	$(GO) vet $(VETPACKAGES)

.PHONY: lint
lint:
	@hash golint > /dev/null 2>&1; if [ $$? -ne 0 ]; then \
		$(GO) get -u golang.org/x/lint/golint; \
	fi
	for PKG in $(PACKAGES); do golint -set_exit_status $$PKG || exit 1; done;

.PHONY: misspell-check
misspell-check:
	@hash misspell > /dev/null 2>&1; if [ $$? -ne 0 ]; then \
		$(GO) get -u github.com/client9/misspell/cmd/misspell; \
	fi
	misspell -error $(GOFILES)

.PHONY: misspell
misspell:
	@hash misspell > /dev/null 2>&1; if [ $$? -ne 0 ]; then \
		$(GO) get -u github.com/client9/misspell/cmd/misspell; \
	fi
	misspell -w $(GOFILES)

.PHONY: tools
tools:
	go install golang.org/x/lint/golint@latest; \
	go install github.com/client9/misspell/cmd/misspell@latest;

.PHONY: migrate
migrate:
	$(GO) test -tags $(TESTTAGS) -run TestModelMigrate$
# ---------------------------------------------------------------------------
# deps: 准备 v8go 原生库（在 Yao-Dev 工作区内复用 ../scripts/v8go-deps.sh）
# 说明: v8go 模块 zip 只含 libv8.a.zip，cgo 链接需要解压后的库；
#       发布版模块可直接解压到模块缓存，未发布版本会使用本地 ../v8go 参考源码。
# ---------------------------------------------------------------------------
deps:
	go run ./scripts/v8godeps
