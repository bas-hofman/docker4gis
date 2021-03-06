#!/bin/bash
set -e

DOCKER_REGISTRY="${DOCKER_REGISTRY}"
DOCKER_USER="${DOCKER_USER}"
DOCKER_TAG="${DOCKER_TAG}"
DOCKER_ENV="${DOCKER_ENV}"
DOCKER_BINDS_DIR="${DOCKER_BINDS_DIR}"

repo=$(basename "$0")
container="${DOCKER_USER}-${repo}"
image="${DOCKER_REGISTRY}${DOCKER_USER}/${repo}:${DOCKER_TAG}"

if .run/start.sh "${image}" "${container}"; then exit; fi

fileport="${DOCKER_BINDS_DIR}/fileport/${DOCKER_USER}"

docker container run --name $container \
	-e DOCKER_USER="${DOCKER_USER}" \
	--network "${DOCKER_USER}" \
	-v "$(docker_bind_source "${fileport}")":/fileport \
	-d $image serve "$@"
