.PHONY: build build-all test test-security clean preview

TEMPLATES := gongwen jiaoan-shicao

# Go 安全黑名单：禁止这些标准库包
GO_STDLIB_DENY := ^net$$|^net/|^os/exec$$|^plugin$$|^debug/

build:
ifndef NAME
	$(error Usage: make build NAME=gongwen)
endif
	cd $(NAME) && go build -trimpath -ldflags="-s -w" -o ../presto-template-$(NAME) .

build-all:
	@for t in $(TEMPLATES); do \
		echo "Building $$t..."; \
		cd $$t && go build -trimpath -ldflags="-s -w" -o ../presto-template-$$t . && cd ..; \
	done

test: build-all test-security
ifndef NAME
	@for t in $(TEMPLATES); do \
		echo "Testing $$t..."; \
		./presto-template-$$t --manifest | python3 -m json.tool > /dev/null && \
		./presto-template-$$t --example | ./presto-template-$$t > /dev/null && \
		./presto-template-$$t --version > /dev/null && \
		echo "  $$t: OK"; \
	done
else
	./presto-template-$(NAME) --manifest | python3 -m json.tool > /dev/null
	./presto-template-$(NAME) --example | ./presto-template-$(NAME) > /dev/null
	./presto-template-$(NAME) --version > /dev/null
endif

test-security:
	@echo "Running security checks..."
	@# 第一层：静态分析（禁止的 import）
	@for t in $(TEMPLATES); do \
		FORBIDDEN=$$(cd $$t && go list -f '{{join .Imports "\n"}}' ./... | grep -E '$(GO_STDLIB_DENY)'); \
		if [ -n "$$FORBIDDEN" ]; then \
			echo "SECURITY FAIL: $$t imports forbidden packages: $$FORBIDDEN"; \
			exit 1; \
		fi; \
	done
	@# 第二层：运行时网络沙箱
	@for t in $(TEMPLATES); do \
		if command -v sandbox-exec >/dev/null 2>&1; then \
			echo "# Test" | sandbox-exec -p '(version 1)(allow default)(deny network*)' ./presto-template-$$t > /dev/null; \
		elif unshare --net true 2>/dev/null; then \
			echo "# Test" | unshare --net ./presto-template-$$t > /dev/null; \
		else \
			echo "SKIP: no sandbox tool available for $$t"; \
		fi; \
	done
	@# 第三层：输出格式验证
	@for t in $(TEMPLATES); do \
		OUTPUT=$$(./presto-template-$$t --example | ./presto-template-$$t); \
		if echo "$$OUTPUT" | grep -qiE '<html|<script|<iframe|<img|<link|<!DOCTYPE'; then \
			echo "SECURITY FAIL: $$t output contains HTML"; \
			exit 1; \
		fi; \
		if ! echo "$$OUTPUT" | head -1 | grep -q '^[#/]'; then \
			echo "SECURITY FAIL: $$t output first line is not a Typst directive"; \
			exit 1; \
		fi; \
	done
	@echo "Security checks passed."

preview:
ifndef NAME
	$(error Usage: make preview NAME=gongwen)
endif
	@mkdir -p ~/.presto/templates/$(NAME)
	cp presto-template-$(NAME) ~/.presto/templates/$(NAME)/
	./presto-template-$(NAME) --manifest > ~/.presto/templates/$(NAME)/manifest.json
	@echo "Installed $(NAME) to ~/.presto/templates/$(NAME)/"

clean:
	rm -f presto-template-*
