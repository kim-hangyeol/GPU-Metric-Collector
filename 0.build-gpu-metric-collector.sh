#!/bin/bash
docker_id="ketidevit3"
image_name="Exascale-keti-gpu-metric-collector"
# image_name="kmc-metric-test"
operator="metric-collector"
version="v0.1"
#version="v284" #82
#khg:W
export GO111MODULE=on
go mod vendor
kubectl config view >> `pwd`/build/bin/config

go build -o `pwd`/build/_output/bin/$operator -mod=vendor `pwd`/cmd/main.go && \
docker build -t $docker_id/$image_name:$version build && \
docker push $docker_id/$image_name:$version
