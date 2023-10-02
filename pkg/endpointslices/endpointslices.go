package endpointslices

import (
	"fmt"

	"context"
	"log"

	corev1 "k8s.io/api/core/v1"
	v1 "k8s.io/api/core/v1"
	discoveryv1 "k8s.io/api/discovery/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	client "github.com/liornoy/main/node-comm-lib/pkg/client"
	"github.com/liornoy/main/node-comm-lib/pkg/consts"
)

func forPod(pod corev1.Pod, slices []discoveryv1.EndpointSlice) (discoveryv1.EndpointSlice, error) {
	for _, slice := range slices {
		for _, endpoint := range slice.Endpoints {
			if endpoint.TargetRef == nil {
				continue
			}
			if endpoint.TargetRef.Name == pod.Name &&
				endpoint.TargetRef.Namespace == pod.Namespace {
				return slice, nil
			}
		}
	}

	return discoveryv1.EndpointSlice{}, fmt.Errorf("failed to find the EndpointSlice for host-networked pod: %s", pod.Name)
}

func forService(service corev1.Service, slices []discoveryv1.EndpointSlice) (discoveryv1.EndpointSlice, error) {
	for _, slice := range slices {
		for _, ownerRef := range slice.OwnerReferences {
			if ownerRef.Name == service.Name {
			}
			return slice, nil
		}
	}

	return discoveryv1.EndpointSlice{}, fmt.Errorf("failed to find the EndpointSlice for service: %s", service.Name)
}

// GetIngressCommSlices reutrn the EndpointSlices in the cluster that are for ingress traffic.
func GetIngressCommSlices(cs *client.ClientSet) ([]discoveryv1.EndpointSlice, error) {
	res := make([]discoveryv1.EndpointSlice, 0)
	slices, err := cs.DiscoveryV1Interface.EndpointSlices("").List(context.TODO(), metav1.ListOptions{})
	if err != nil {
		return nil, err
	}

	withHostNetwork, err := withHostNetwork(cs, slices.Items)
	if err != nil {
		return nil, err
	}
	res = append(res, withHostNetwork...)

	withIngressLabel := withIngressLabel(cs, slices.Items)
	res = append(res, withIngressLabel...)

	withIngressService, err := withIngressService(cs, slices.Items)
	if err != nil {
		return nil, err
	}
	res = append(res, withIngressService...)

	return res, nil
}

// withHostNetwrok filters slices that belongs to host-networked pods
func withHostNetwork(cs *client.ClientSet, slices []discoveryv1.EndpointSlice) ([]discoveryv1.EndpointSlice, error) {
	res := make([]discoveryv1.EndpointSlice, 0)
	pods, err := cs.CoreV1Interface.Pods("").List(context.TODO(), metav1.ListOptions{})
	if err != nil {
		return nil, err
	}

	for _, pod := range pods.Items {
		if pod.Spec.HostNetwork == false {
			continue
		}
		slice, err := forPod(pod, slices)
		if err != nil {
			// log the error
			log.Print(err)
			continue
		}
		res = append(res, slice)
	}

	return res, nil
}

// withIngressLabel filters slices that belong to the host (filtered via label=ingress)
func withIngressLabel(cs *client.ClientSet, slices []discoveryv1.EndpointSlice) []discoveryv1.EndpointSlice {
	res := make([]discoveryv1.EndpointSlice, 0)
	for _, slice := range slices {
		if _, ok := slice.Labels[consts.IngressLabel]; ok {
			res = append(res, slice)
		}
	}

	return res
}

// withIngressService filters slices of services with type NodePort|LoadBalancer
func withIngressService(cs *client.ClientSet, slices []discoveryv1.EndpointSlice) ([]discoveryv1.EndpointSlice, error) {
	res := make([]discoveryv1.EndpointSlice, 0)
	services, err := cs.CoreV1Interface.Services("").List(context.TODO(), metav1.ListOptions{})
	if err != nil {
		return nil, err
	}

	for _, service := range services.Items {
		if service.Spec.Type != v1.ServiceTypeNodePort &&
			service.Spec.Type != v1.ServiceTypeLoadBalancer {
			continue
		}

		slice, err := forService(service, slices)
		if err != nil {
			return nil, err
		}
		res = append(res, slice)
	}

	return res, nil
}
