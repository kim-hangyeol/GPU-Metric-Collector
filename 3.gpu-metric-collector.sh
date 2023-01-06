#!/usr/bin/env bash

#$1 create/c or delete/d

if [ "$1" == "delete" ] || [ "$1" == "d" ]; then   
    echo kubectl delete -f deployments/gpu-metric-collector.yaml
    kubectl delete -f deployments/gpu-metric-collector.yaml
else
    echo kubectl create -f deployments/gpu-metric-collector.yaml
    kubectl create -f deployments/gpu-metric-collector.yaml
fi