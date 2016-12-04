ifdef CANONICAL_SKSSPIDER_REPO
REPO_PATH=      $(CANONICAL_SKSSPIDER_REPO)
else
REPO_PATH=      github.com/philpennock/sks_spider
endif

GO_CMD ?= go
GO_LDFLAGS:=
BUILD_TAGS:=

ifndef REPO_VERSION
REPO_VERSION := $(shell ./.version)
endif
GO_LDFLAGS+= -X $(REPO_PATH).VersionString=$(REPO_VERSION)

SOURCES=       $(shell find . -name vendor -prune -o -type f -name '*.go')
BIN_DIR_TOP := $(firstword $(subst :, ,$(GOPATH)))/bin
BINARY_NAME := sks_stats_daemon
BINARY_DIR  := cmd/$(BINARY_NAME)
BINARY_SRC  := cmd/$(BINARY_NAME)/main.go

.PHONY : all clean install
.DEFAULT_GOAL := all

all: $(BINARY_DIR)/$(BINARY_NAME)

$(BINARY_DIR)/$(BINARY_NAME): $(BINARY_SRC) $(SOURCES)
ifeq ($(REPO_VERSION),)
	@echo "Missing a REPO_VERSION"
	@false
endif
	@echo "Building version $(REPO_VERSION) ..."
	$(GO_CMD) build -o $@ -tags "$(BUILD_TAGS)" -ldflags "$(GO_LDFLAGS)" -v $<

install: $(BINARY_SRC) $(SOURCES)
ifeq ($(REPO_VERSION),)
	@echo "Missing a REPO_VERSION"
	@false
endif
	@echo "Installing version $(REPO_VERSION) ..."
	rm -f "$(BIN_DIR_TOP)/$(BINARY_NAME)"
	$(GO_CMD) install -tags "$(BUILD_TAGS)" -ldflags "$(GO_LDFLAGS)" -v $(REPO_PATH)/...

clean:
	rm -fv $(BINARY_DIR)/$(BINARY_NAME)
