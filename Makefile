# SPDX-FileCopyrightText: 2024-2025 Rafael V. Volkmer <rafael.v.volkmer@gmail.com>
# SPDX-License-Identifier: MIT

BINARY := codeaudit
ANALYZE_PATH ?= .

.PHONY: build test lint run

build:
	go build -o bin/$(BINARY) ./cmd/codeaudit

test:
	go test ./...

lint:
	go vet ./...

run:
	go run ./cmd/codeaudit analyze $(ANALYZE_PATH)
