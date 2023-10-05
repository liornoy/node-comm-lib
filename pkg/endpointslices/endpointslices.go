package endpointslices

import (
	"context"
	"fmt"

	corev1 "k8s.io/api/core/v1"
	discoveryv1 "k8s.io/api/discovery/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	client "github.com/liornoy/main/node-comm-lib/pkg/client"
)

type QueryBuilder interface {
	Query() []discoveryv1.EndpointSlice
	WithLabels(labels map[string]string) QueryBuilder
	WithHostNetwork() QueryBuilder
	WithServiceType(serviceType corev1.ServiceType) QueryBuilder
}

type QueryParams struct {
	cs       *client.ClientSet
	pods     []corev1.Pod
	filter   []bool
	epSlices []discoveryv1.EndpointSlice
	services []corev1.Service
}

func NewQuery(cs *client.ClientSet, namespace string) (*QueryParams, error) {
	if cs == nil {
		return nil, fmt.Errorf("failed to create QueryParams: clientset is nil")
	}

	epSlicesList, err := cs.DiscoveryV1Interface.EndpointSlices(namespace).List(context.TODO(), metav1.ListOptions{})
	if err != nil {
		return nil, fmt.Errorf("failed to create QueryParams: %w", err)
	}

	servicesList, err := cs.CoreV1Interface.Services(namespace).List(context.TODO(), metav1.ListOptions{})
	if err != nil {
		return nil, fmt.Errorf("failed to create QueryParams: %w", err)
	}

	podsList, err := cs.CoreV1Interface.Pods(namespace).List(context.TODO(), metav1.ListOptions{})
	if err != nil {
		return nil, fmt.Errorf("failed to create QueryParams: %w", err)
	}

	ret := QueryParams{
		cs:       cs,
		epSlices: epSlicesList.Items,
		services: servicesList.Items,
		pods:     podsList.Items,
		filter:   make([]bool, len(epSlicesList.Items))}

	return &ret, nil
}

func (q *QueryParams) Query() []discoveryv1.EndpointSlice {
	ret := make([]discoveryv1.EndpointSlice, 0)

	for i, filter := range q.filter {
		if filter {
			ret = append(ret, q.epSlices[i])
		}
	}

	return ret
}

func (q *QueryParams) WithLabels(labels map[string]string) QueryBuilder {
	for i, epSlice := range q.epSlices {
		if q.withLabels(epSlice, labels) {
			q.filter[i] = true
		}
	}

	return q
}

func (q *QueryParams) WithHostNetwork() QueryBuilder {
	for i, epSlice := range q.epSlices {
		if q.withHostNetworked(epSlice) {
			q.filter[i] = true
		}
	}

	return q
}

func (q *QueryParams) WithServiceType(serviceType corev1.ServiceType) QueryBuilder {
	for i, epSlice := range q.epSlices {
		if q.withServiceType(epSlice, serviceType) {
			q.filter[i] = true
		}
	}

	return q
}

func (q *QueryParams) withLabels(epSlice discoveryv1.EndpointSlice, labels map[string]string) bool {
	for key, value := range labels {
		if mValue, ok := epSlice.Labels[key]; !ok || mValue != value {
			return false
		}
	}

	return true
}

func (q *QueryParams) withServiceType(epSlice discoveryv1.EndpointSlice, serviceType corev1.ServiceType) bool {
	if len(epSlice.OwnerReferences) == 0 {
		return false
	}

	for _, ownerRef := range epSlice.OwnerReferences {
		name := ownerRef.Name
		namespace := epSlice.Namespace
		service := getService(name, namespace, q.services)
		if service == nil {
			continue
		}
		if service.Spec.Type == serviceType {
			return true
		}
	}

	return false
}

func (q *QueryParams) withHostNetworked(epSlice discoveryv1.EndpointSlice) bool {
	if len(epSlice.Endpoints) == 0 {
		return false
	}

	for _, endpoint := range epSlice.Endpoints {
		if endpoint.TargetRef == nil {
			continue
		}
		name := endpoint.TargetRef.Name
		namespace := endpoint.TargetRef.Namespace
		pod := getPod(name, namespace, q.pods)
		if pod == nil {
			continue
		}
		if pod.Spec.HostNetwork {
			return true
		}
	}

	return false
}

func getPod(name, namespace string, pods []corev1.Pod) *corev1.Pod {
	for i, p := range pods {
		if p.Name == name && p.Namespace == namespace {
			return &pods[i]
		}
	}

	return nil
}

func getService(name, namespace string, services []corev1.Service) *corev1.Service {
	for i, service := range services {
		if service.Name == name && service.Namespace == namespace {
			return &services[i]
		}
	}

	return nil
}
