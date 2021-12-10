# в мэйке 4 вирт цели: run - сборка и запуск/ build - сборка бинарника/ linters - запуск линтеров
# и pre-commit по конфигурации

# build and run
.PHONY: run
run:
	go run ./lesson7/main.go

# build binary
.PHONY: build
build: test lint
	go build -o mystupidbin ./lesson7/main.go

# run linters
.PHONY: lint
lint:
	golangci-lint run ./...
	pre-commit run --verbose

# pre-commit hooks
.PHONY: pre-commit
pre-commit:
	pre-commit install

