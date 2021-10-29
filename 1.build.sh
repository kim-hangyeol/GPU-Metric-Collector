#!/bin/bash
docker_id="ketidevit"
image_name="kmc-metric-test"
operator="metric-collector"
version=v119 #82

export GO111MODULE=on
go mod vendor
kubectl config view >> `pwd`/build/bin/config

go build -o `pwd`/build/_output/bin/$operator -mod=vendor `pwd`/cmd/main.go && \
docker build -t $docker_id/$image_name:$version build && \
docker push $docker_id/$image_name:$version
