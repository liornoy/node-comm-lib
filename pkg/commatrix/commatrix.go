package commatrix

import (
	"context"
	"fmt"
	"os"
	"strings"
	"text/tabwriter"

	discoveryv1 "k8s.io/api/discovery/v1"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/liornoy/main/node-comm-lib/pkg/client"
	"github.com/liornoy/main/node-comm-lib/pkg/consts"
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

func (cd CommDetails) String() string {
	return fmt.Sprintf("%s\t\t\t%s\t\t\t%s\t\t\t%s\t\t\t%s", cd.Direction, cd.NodeRole, cd.Protocol, cd.Port, cd.ServiceName)
}

func (m CommMatrix) PrintMat() {
	w := new(tabwriter.Writer)
	w.Init(os.Stdout, 8, 8, 0, '\t', 0)
	defer w.Flush()
	fmt.Fprintf(w, " %s\t\t%s\t\t%s\t\t%s\t\t%s\n", "DIRECTION", "NODE-ROLE", "PROTOCOL", "PORT", "SERVICE")

	for i, cd := range m.Matrix {
		fmt.Fprintf(w, "%d. %s\n", i+1, cd)
	}
}

func CreateCommMatrix(cs *client.ClientSet, slices []discoveryv1.EndpointSlice) (CommMatrix, error) {
	res := make([]CommDetails, 0)

	nodesRoles, err := GetNodesRoles(cs)
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
	res = RemoveDups(res)

	return CommMatrix{Matrix: res}, nil
}

func RemoveDups(outPuts []CommDetails) []CommDetails {
	allKeys := make(map[string]bool)
	res := []CommDetails{}
	for _, item := range outPuts {
		str := fmt.Sprintf("%s-%s-%s", item.NodeRole, item.Port, item.Protocol)
		if _, value := allKeys[str]; !value {
			allKeys[str] = true
			res = append(res, item)
		}
	}

	return res
}

func GetNodesRoles(cs *client.ClientSet) (map[string]string, error) {
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
