#!/bin/bash
PODNAME=""
while [ -z $PODNAME ]
do
    PODNAME=`kubectl get po -o=name -A --field-selector=status.phase=Running | grep keti-gpu-metric-collector`
    PODNAME="${PODNAME:4}"
done
echo 
sleep 1
echo "--------------------:: KETI GPU Metric Collector Log ::--------------------"
echo "---------------------------------------------------------------------------"
sleep 1
kubectl logs $PODNAME -n gpu -f --tail=19