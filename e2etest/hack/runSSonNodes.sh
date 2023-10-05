#!/bin/sh

mkdir artifacts

PODS=$(oc get pods -n openshift-machine-config-operator -o name | grep machine-config-daemon)

for POD in ${PODS}; do
    NODE=$(oc get "${POD}" -n openshift-machine-config-operator -o 'jsonpath={.spec.nodeName}')
    oc exec "${POD}" -n openshift-machine-config-operator -c machine-config-daemon -- ss -plant > ./artifacts/"${NODE}"-tcp.txt
    oc exec "${POD}" -n openshift-machine-config-operator -c machine-config-daemon -- ss -planu > ./artifacts/"${NODE}"-udp.txt
done
