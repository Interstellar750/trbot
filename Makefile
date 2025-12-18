COMMIT   := $(shell git rev-parse HEAD)
BRANCH   := $(shell git rev-parse --abbrev-ref HEAD)
VERSION  := $(shell git describe --tags --always)
ifeq ($(OS),Windows_NT)
    TIME    := $(shell powershell -Command "[int][double]::Parse((Get-Date -UFormat %s))")
    CHANGES := $(shell powershell -NoProfile -Command "(git status -s | Measure-Object).Count")
else
    TIME    := $(shell date +%s)
    CHANGES := $(shell git status -s | wc -l)
endif
HOSTNAME := $(shell hostname)
LDFLAGS  := -X 'trle5.xyz/gopkg/trbot/utils/configs.Commit=$(COMMIT)' \
            -X 'trle5.xyz/gopkg/trbot/utils/configs.Branch=$(BRANCH)' \
            -X 'trle5.xyz/gopkg/trbot/utils/configs.Version=$(VERSION)' \
            -X 'trle5.xyz/gopkg/trbot/utils/configs.Changes=$(CHANGES)' \
            -X 'trle5.xyz/gopkg/trbot/utils/configs.BuildAt=$(TIME)' \
            -X 'trle5.xyz/gopkg/trbot/utils/configs.BuildOn=$(HOSTNAME)'

build:
	go build -ldflags "$(LDFLAGS)"
