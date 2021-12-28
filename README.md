# KETI-ExaScale GPU-Metric-Collector
## Introduction of KETI-ExaScale GPU-Metric-Collector
GPU-Metric-Collector for KETI-ExaScale Platform

Developed by KETI
## 목차
[1. Requirment](#requirement)

[2. How to Install](#how-to-install)

[3. Install Check](#install-check)

[4. Governance](#governance)

## Requirement
> Kubernetes <= 1.24

> InfluxDB

> KETI-ExaScale GPU-Scheduler

> KETI-ExaScale GPU-Device-Plugin
## How to Install
    $ kubectl apply -f metricgpu.yaml
## Install Check
Create Check

    $ kubectl get pods -A
    NAMESPACE     NAME                                  READY   STATUS      RESTARTS      AGE
    gpu           keti-gpu-metric-collector-h6l2l       1/1     Running     0             21s
Log Check

    $ kubectl logs [keti-gpu-metric-collector] -n gpu
    2021/12/27 07:45:38 start gRPC server on 9000 port
## Governance
> This work was supported by Institute of Information & communications Technology Planning & Evaluation (IITP) grant funded by the Korea government(MSIT) (No.2021-0-00862, Development of DBMS storage engine technology to minimize massive data movement)