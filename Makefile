BIN := cmd/file_server

GITTAG := `git describe --tags`
VERSION := `git describe --abbrev=0 --tags`
RELEASE := `git rev-list $(shell git describe --abbrev=0 --tags).. --count`
BUILD_TIME := `date +%FT%T%z`
# Setup the -ldflags option for go build here, interpolate the variable values
LDFLAGS := -ldflags "-X main.GitTag=${GITTAG} -X main.BuildTime=${BUILD_TIME}"

vendor:
	go get ./...
	go get github.com/stretchr/testify/assert


test:
	go list ./... |\
	while IFS= read -r line ; \
	do \
		go test ""$$line"" -cover; \
	done

race:
	go test -v ./... --race -cover;

fmt:
	find . -name "*.go" -not -path "./vendor/*" -type f -exec echo {} \;  |\
	while IFS= read -r line; \
	do \
		echo "$$line";\
		goimports -w "$$line" "$$line";\
	done

build:
	mkdir -p bin;\
	echo ==================================; \
	for m in $(BIN); do \
		cd $(PWD)/$$m && go build ${LDFLAGS} -o ../../bin/$$(basename $$m)  ; \
	done
	echo ==================================; \

install: vendor test build
