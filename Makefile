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
LDFLAGS  := -X 'trbot/utils/configs.Commit=$(COMMIT)' \
            -X 'trbot/utils/configs.Branch=$(BRANCH)' \
            -X 'trbot/utils/configs.Version=$(VERSION)' \
            -X 'trbot/utils/configs.Changes=$(CHANGES)' \
            -X 'trbot/utils/configs.BuildAt=$(TIME)' \
            -X 'trbot/utils/configs.BuildOn=$(HOSTNAME)'

build:
	go build -ldflags "$(LDFLAGS)"
