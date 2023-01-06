#!/bin/bash
PODNAMES=""
while [ -z $PODNAMES ]
do
    PODNAMES=`kubectl get pod -A -o=name -o=wide --field-selector=status.phase=Running | grep keti-gpu-metric-collector | grep gpu-server`
done

LIST=($PODNAMES)
PODNAME=${LIST[1]}
# echo PODNAME

echo 
sleep 1
echo "--------------------:: KETI GPU Metric Collector Log ::--------------------"
echo "---------------------------------------------------------------------------"
sleep 1
kubectl logs $PODNAME -n gpu --tail=19