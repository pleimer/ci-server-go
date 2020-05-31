#!/bin/sh

podman build -t ci-server-go .
podman tag ci-server-go quay.io/plmr/ci-server-go
podman push quay.io/plmr/ci-server-go

