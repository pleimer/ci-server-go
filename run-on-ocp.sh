#!/bin/bash

sed "s/<oauth_token>/$(cat ~/accessgit)/g" deploy-on-k8s.yml > /tmp/ci-ocp.yml
oc create -f /tmp/ci-ocp.yml
