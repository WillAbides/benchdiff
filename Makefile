GOCMD=go
GOBUILD=$(GOCMD) build
PATH := "${CURDIR}/bin:$(PATH)"

.PHONY: gobuildcache

bin/golangci-lint:
	script/bindown install $(notdir $@)

bin/shellcheck:
	script/bindown install $(notdir $@)

bin/gobin:
	script/bindown install $(notdir $@)

HANDCRAFTED_REV := 082e94edadf89c33db0afb48889c8419a2cb46a9
bin/handcrafted: bin/gobin
	GOBIN=${CURDIR}/bin \
	bin/gobin github.com/willabides/handcrafted@$(HANDCRAFTED_REV)

GOFUMPT_REV := 4fd085cb6d5fb7ec2bb2c6fc8039ec3a48355807
bin/gofumpt: bin/gobin
	GOBIN=${CURDIR}/bin \
	bin/gobin mvdan.cc/gofumpt@$(GOFUMPT_REV)
