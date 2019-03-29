#!/bin/bash

mkdir -p tmp


VERSION="1.11.1"
OS="linux"
ARCH="amd64"
FILE="go$VERSION.$OS-$ARCH.tar.gz"

if [ ! -f "tmp/${FILE}" ]; then
    echo "Downloading go ${VERSION}"
    pushd tmp
    wget "https://dl.google.com/go/${FILE}"
    tar -xzf "${FILE}"
    popd
fi

export GO111MODULE=on

echo "Build pprof-exporter"

CGO_ENABLED=0 go build -a -ldflags '-extldflags "-static"' -o tmp/pprof-exporter cmd/pprof-exporter/main.go

docker build -t sbueringer/pprof-exporter -f Dockerfile.exporter .

echo "Build pprof-importer"

CGO_ENABLED=0 go build -a -ldflags '-extldflags "-static"' -o tmp/pprof-importer cmd/pprof-importer/main.go

docker build -t sbueringer/pprof-importer -f Dockerfile.importer .
