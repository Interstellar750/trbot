COMMIT   := $(shell git rev-parse HEAD)
BRANCH   := $(shell git rev-parse --abbrev-ref HEAD)
VERSION  := $(shell git describe --tags --always)
ifeq ($(OS),Windows_NT)
    TIME    := $(shell powershell -Command "Get-Date -UFormat %s")
    CHANGES := $(shell powershell -NoProfile -Command "(git status -s | Measure-Object).Count")
else
    TIME    := $(shell date +%s)
    CHANGES := $(shell git status -s | wc -l)
endif
HOSTNAME := $(shell hostname)
LDFLAGS  := -X 'trbot/utils/consts.Commit=$(COMMIT)' \
            -X 'trbot/utils/consts.Branch=$(BRANCH)' \
            -X 'trbot/utils/consts.Version=$(VERSION)' \
            -X 'trbot/utils/consts.Changes=$(CHANGES)' \
            -X 'trbot/utils/consts.BuildAt=$(TIME)' \
            -X 'trbot/utils/consts.BuildOn=$(HOSTNAME)'

build:
	go build -ldflags "$(LDFLAGS)"
