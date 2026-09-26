#!/usr/bin/env bash

set -euo pipefail

log() {
    echo "build-image.sh: $*"
}

name="ovirt-csi-driver"
version="${OVIRT_CSI_DRIVER_VERSION:-${VERSION:-${1:-}}}"
registry="container-registry.oracle.com/olcne"
docker_tag="${registry}/${name}:v${version}"

if [[ -z "${version}" ]]; then
    log "unable to determine oVirt CSI driver version; set OVIRT_CSI_DRIVER_VERSION or pass it as the first argument" >&2
    exit 1
fi

build_args=(
    --pull
    --from container-registry.oracle.com/os/oraclelinux:9-slim
    -t "${docker_tag}"
    -f ./olm/builds/Dockerfile
    .
)

if [[ -n "${https_proxy:-}" ]]; then
    build_args=(
        --build-arg "https_proxy=${https_proxy}"
        "${build_args[@]}"
    )
fi

mount_yum_config() {
    local yum_repo_config_file="${YUM_REPO_CONFIG_FILE:-}"

    if [[ -z "${yum_repo_config_file}" ]]; then
        log "YUM_REPO_CONFIG_FILE is not set; using base image repository configuration"
        return
    elif [[ "${yum_repo_config_file}" != /* ]]; then
        yum_repo_config_file="$(pwd)/${yum_repo_config_file}"
    fi

    log "checking yum repo config file ${yum_repo_config_file}"
    if [[ ! -s "${yum_repo_config_file}" ]]; then
        log "yum repo config file ${yum_repo_config_file} is missing or empty" >&2
        exit 1
    fi

    log "mounting yum repo config file ${yum_repo_config_file}"
    build_args=(
        --volume "${yum_repo_config_file}:/etc/yum.repos.d/extra.repo:ro"
        "${build_args[@]}"
    )
}

mount_yum_config

log "building ${docker_tag}"
podman build "${build_args[@]}"

log "saving ${docker_tag} to ${name}.tar"
podman save -o "${name}.tar" "${docker_tag}"
