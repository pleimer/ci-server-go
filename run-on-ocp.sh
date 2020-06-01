#!/bin/bash

if [[ -z "${GITHUB_USER}" ]]; then 
    echo "GITHUB_USER must be specified"
    exit -1
fi


if [[ -z "${GITHUB_OAUTH}" ]]; then 
    echo "GITHUB_OAUTH must be specified"
    exit -1
fi

sed "s/<oauth_token>/$(cat ~/accessgit)/g" deploy-on-k8s.yml > /tmp/ci-ocp.yml
sed "s/<github_user>/${GITHUB_USER}/g" /tmp/ci-ocp.yml > /tmp/ci-ocp.yml
oc create -f /tmp/ci-ocp.yml
