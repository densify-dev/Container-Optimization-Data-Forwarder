#!/bin/bash

exec=$(basename "$0")

usage() {
    echo "" >&2
    echo "usage: ${exec} [ -b baseImage ] [ -t imageTag ] [ -p ] [ -r ] [ -h ]" >&2
    echo "" >&2
    echo "  b - alpine, ubi9, debian [ default is alpine ] " >&2
    echo "  t - required image tag [ mandatory ] " >&2
    echo "  o - official release image (implied tagging) " >&2
    echo "  p - tag & push image to quay.io and Docker hub " >&2
    echo "  h - print help and exit " >&2
    echo "" >&2

    exit $1
}

tagAndPush() {
    action="tag $1 $2"
    docker ${action}
    rc=$?
    if [ $rc -eq 0 ]; then
        action="push $2"
        docker ${action}
        rc=$?
    fi
    if [ $rc -ne 0 ]; then
        echo "docker ${action} failed with return code $rc, exiting"
        exit $rc
    fi
}

gitCommitHash() {
    git rev-parse --verify HEAD
    rc=$?
    if [ $rc -ne 0 ]; then
        echo "failed to get git commit hash, exiting"
        exit $rc
    fi
}

baseImageArg="alpine"
tag=""
push=0
official=0

while getopts 'b:t:oph' opt; do
    case $opt in
    # general options
    b) baseImageArg=$OPTARG ;;
    t) tag=$OPTARG ;;
    o) official=1 ;;
    p) push=1 ;;
    # user asked for help, only case usage is called with 0
    h) usage 0 ;;
    # wrong option - usage error
    *) usage 1 ;;
    esac
done

if [ -z "${tag}" ]; then
    usage 1
fi

# full name of ubi9 image
if [ "${baseImageArg}" == "ubi9" ]; then
    baseImage="registry.access.redhat.com/ubi9/ubi-minimal"
else
    baseImage="${baseImageArg}"
fi

quayImage="container-data-collection-forwarder"
quayRepo="quay.io/densify/"
dockerHubImage="container-optimization-data-forwarder"
dockerHubRepo="densify/"

release=$(gitCommitHash)

# build the image
docker build --progress=plain -t ${quayImage}:${baseImageArg}-${tag} -f Dockerfile --build-arg BASE_IMAGE=${baseImage} --build-arg VERSION=${tag} --build-arg RELEASE=${release} .
# use docker login w/ credentials to login to quay.io
# use docker login w/ credentials to login to Docker hub (no server specified)
if [ ${push} -eq 1 ]; then
    tagAndPush ${quayImage}:${baseImageArg}-${tag} ${quayRepo}${quayImage}:${baseImageArg}-${tag}
    if [ "${baseImageArg}" == "alpine" ]; then
        tagAndPush ${quayImage}:${baseImageArg}-${tag} ${dockerHubRepo}${dockerHubImage}:${baseImageArg}-${tag}
    fi
    if [ ${official} -eq 1 ]; then
        tagAndPush ${quayImage}:${baseImageArg}-${tag} ${quayRepo}${quayImage}:${baseImageArg}
        if [ "${baseImageArg}" == "alpine" ]; then
            tagAndPush ${quayImage}:${baseImageArg}-${tag} ${quayRepo}${quayImage}:latest
            tagAndPush ${quayImage}:${baseImageArg}-${tag} ${dockerHubRepo}${dockerHubImage}:${baseImageArg}
            tagAndPush ${quayImage}:${baseImageArg}-${tag} ${dockerHubRepo}${dockerHubImage}:latest
        fi
    fi
fi
