#!/bin/bash

REGISTRY=${1:-"docker.io/sbueringer"}
DOCKER_CONFIG=${2:="${HOME}/docker"}

docker tag sbueringer/pprof-exporter "${REGISTRY}/pprof-exporter"
docker --config "${DOCKER_CONFIG}" push "${REGISTRY}/pprof-exporter"

docker tag sbueringer/pprof-importer "${REGISTRY}/pprof-importer"
docker --config "${DOCKER_CONFIG}" push "${REGISTRY}/pprof-importer"
