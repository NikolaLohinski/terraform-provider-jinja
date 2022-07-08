default: build

build:
	go build -v ./...

lint:
	golangci-lint run

install:
	go build -o ${HOME}/.terraform.d/plugins/nikolalohinski/jinja/1.0.0/linux_amd64/terraform-provider-jinja

fmt:
	gofmt -s -w -e .

test:
	go test -v -cover -timeout=120s -parallel=4 ./...

.PHONY: build install lint fmt test
