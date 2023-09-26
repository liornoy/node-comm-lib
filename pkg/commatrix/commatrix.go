package commatrix

import (
	"context"
	"fmt"
	"strings"

	"github.com/liornoy/main/node-comm-lib/pkg/client"
	"github.com/liornoy/main/node-comm-lib/pkg/consts"
	discoveryv1 "k8s.io/api/discovery/v1"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

type CommMatrix struct {
	Matrix []CommDetails
}

type CommDetails struct {
	Direction   string `json:"direction"`
	Protocol    string `json:"protocol"`
	Port        string `json:"port"`
	NodeRole    string `json:"nodeRole"`
	ServiceName string `json:"serviceName"`
}

func CreateCommMatrix(cs *client.ClientSet, slices []discoveryv1.EndpointSlice) (CommMatrix, error) {
	res := make([]CommDetails, 0)

	nodesRoles, err := getNodesRoles(cs)
	if err != nil {
		return CommMatrix{}, err
	}

	for _, slice := range slices {
		ports := make([]string, 0)
		protocols := make([]string, 0)
		services := make([]string, 0)
		for _, p := range slice.Ports {
			ports = append(ports, fmt.Sprint(*p.Port))
			protocols = append(protocols, fmt.Sprint(*p.Protocol))
		}
		for _, ownerRed := range slice.OwnerReferences {
			services = append(services, ownerRed.Name)
		}
		for _, endpoint := range slice.Endpoints {
			commDetails := &CommDetails{
				Direction:   "ingress",
				Protocol:    strings.Join(protocols, ","),
				Port:        strings.Join(ports, ","),
				NodeRole:    nodesRoles[*endpoint.NodeName],
				ServiceName: strings.Join(services, ","),
			}
			res = append(res, *commDetails)
		}
	}

	return CommMatrix{Matrix: res}, nil
}

func getNodesRoles(cs *client.ClientSet) (map[string]string, error) {
	res := make(map[string]string)
	nodes, err := cs.Nodes().List(context.TODO(), v1.ListOptions{})
	if err != nil {
		return nil, err
	}

	for _, node := range nodes.Items {
		_, isWorker := node.Labels[consts.WorkerRole]
		_, isMaster := node.Labels[consts.MasterRole]
		if isMaster && isWorker {
			res[node.Name] = "master-worker"
			continue
		}
		if isMaster {
			res[node.Name] = "master"
		}
		if isWorker {
			res[node.Name] = "worker"
		}
	}

	return res, nil
}
