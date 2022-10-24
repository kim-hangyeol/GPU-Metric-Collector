# GPU-Metric-Collector

## GPU-Metric-Collector 소개

## 목차
[1. 필요 환경](#필요-환경)

[2. 설치 방법](#설치-방법)

[3. 설치 확인](#설치-확인)

## 필요 환경
> 쿠버네티스 <= 1.24

> InfluxDB

> KETI GPU Scheduler

> KETI GPU Device Plugin
## 설치 방법
    $ kubectl apply -f metricgpu.yaml

## 설치 확인
생성 확인

    $ kubectl get pods -A
    NAMESPACE     NAME                                  READY   STATUS      RESTARTS      AGE
    gpu           keti-gpu-metric-collector-h6l2l       1/1     Running     0             21s

로그 확인

    $ kubectl logs [keti-gpu-metric-collector] -n gpu
    2021/12/27 07:45:38 start gRPC server on 9000 port