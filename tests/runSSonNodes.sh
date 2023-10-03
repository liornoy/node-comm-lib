#!/bin/sh

mkdir artifacts

PODS=$(oc get pods -n openshift-machine-config-operator -o name | grep machine-config-daemon)

for POD in ${PODS}; do
    NODE=$(oc get "${POD}" -n openshift-machine-config-operator -o 'jsonpath={.spec.nodeName}')
    oc exec -it "${POD}" -n openshift-machine-config-operator -- ss -plant > ./artifacts/"${NODE}"-tcp.txt
    oc exec -it "${POD}" -n openshift-machine-config-operator -- ss -planu > ./artifacts/"${NODE}"-udp.txt
done
