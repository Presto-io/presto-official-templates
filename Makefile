.PHONY: build build-all test clean preview

TEMPLATES := gongwen jiaoan-shicao

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

test:
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
