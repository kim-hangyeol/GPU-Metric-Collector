#!/usr/bin/env bash

#$1 create/c or delete/d

if [ "$1" == "delete" ] || [ "$1" == "d" ]; then   
    echo kubectl delete -f deployments/gpu-metric-collector-role-binding.yaml
    echo kubectl delete -f deployments/gpu-metric-collector-service-account.yaml
    kubectl delete -f deployments/gpu-metric-collector-role-binding.yaml
    kubectl delete -f deployments/gpu-metric-collector-service-account.yaml
else
    echo kubectl create -f deployments/gpu-metric-collector-role-binding.yaml
    echo kubectl create -f deployments/gpu-metric-collector-service-account.yaml
    kubectl create -f deployments/gpu-metric-collector-role-binding.yaml
    kubectl create -f deployments/gpu-metric-collector-service-account.yaml
fi
