#!/bin/bash
set -e

echo "Installing Go analysis tools..."

go install honnef.co/go/tools/cmd/staticcheck@latest
go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest
go install github.com/securego/gosec/v2/cmd/gosec@latest
go install github.com/kisielk/errcheck@latest
go install github.com/fatih/structtag/cmd/structtag@latest
go install mvdan.cc/unparam@latest
go install github.com/gordonklaus/ineffassign@latest
go install golang.org/x/tools/cmd/goimports@latest

go install golang.org/x/perf/cmd/benchstat@latest
go install github.com/mgechev/revive@latest
go install github.com/sasha-s/go-deadlock@latest
go install github.com/go-delve/delve/cmd/dlv@latest # see: https://github.com/go-delve/delve?tab=readme-ov-file

echo 'export PATH="$PATH:$(go env GOPATH)/bin"' >> ~/.bashrc && source ~/.bashrc

echo "Installing complete."
