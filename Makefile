VERSION  := $(shell git describe --tags --always)
COMMIT   := $(shell git rev-parse HEAD)
CHANGES  := $(shell git status -s | wc -l)
TIME     := $(shell date --rfc-3339=seconds)
HOSTNAME := $(shell hostname)
LDFLAGS  := -X 'trbot/utils/consts.Version=$(VERSION)' \
            -X 'trbot/utils/consts.Commit=$(COMMIT)' \
            -X 'trbot/utils/consts.Changes=$(CHANGES)' \
            -X 'trbot/utils/consts.BuildTime=$(TIME)' \
            -X 'trbot/utils/consts.BuildMachine=$(HOSTNAME)'

build:
	go build -ldflags "$(LDFLAGS)"
