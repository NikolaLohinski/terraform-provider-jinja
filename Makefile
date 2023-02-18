default: build

build:
	go build -v ./...

lint:
	golangci-lint run

install:
	go build -o /tmp/terraform-provider-jinja/terraform-provider-jinja

fmt:
	gofmt -s -w -e ./provider ./lib

test:
	go test -v -cover -timeout=120s -parallel=4 ./...

example: install
	TF_CLI_CONFIG_FILE=$(PWD)/development.tfrc terraform -chdir=$(PWD)/examples apply

.PHONY: build install lint fmt test
