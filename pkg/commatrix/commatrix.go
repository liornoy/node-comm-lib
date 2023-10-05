package commatrix

import (
	"bytes"
	"context"
	"encoding/csv"
	"fmt"
	"strings"

	corev1 "k8s.io/api/core/v1"
	discoveryv1 "k8s.io/api/discovery/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/liornoy/main/node-comm-lib/pkg/client"
	"github.com/liornoy/main/node-comm-lib/pkg/consts"
)

type ComMatrix struct {
	Matrix []ComDetails
}

type ComDetails struct {
	Direction   string `json:"direction"`
	Protocol    string `json:"protocol"`
	Port        string `json:"port"`
	NodeRole    string `json:"nodeRole"`
	ServiceName string `json:"serviceName"`
	Required    string `json:"required"`
}

func (cd ComDetails) String() string {
	return fmt.Sprintf("%s,%s,%s,%s,%s,%s", cd.Direction, cd.Protocol, cd.Port, cd.NodeRole, cd.ServiceName, cd.Required)
}

func CreateComMatrix(cs *client.ClientSet, epSlices []discoveryv1.EndpointSlice) (ComMatrix, error) {
	if len(epSlices) == 0 {
		return ComMatrix{}, fmt.Errorf("failed to create ComMatrix: epSlices is empty")
	}

	nodes, err := cs.Nodes().List(context.TODO(), metav1.ListOptions{})
	if err != nil {
		return ComMatrix{}, fmt.Errorf("failed to create ComMatrix: %w", err)
	}

	nodesRoles := GetNodesRoles(nodes)
	res := make([]ComDetails, 0)

	for _, epSlice := range epSlices {
		required := "true"
		if _, ok := epSlice.Labels["optional"]; ok {
			required = "false"
		}
		ports := make([]string, 0)
		protocols := make([]string, 0)
		for _, p := range epSlice.Ports {
			ports = append(ports, fmt.Sprint(*p.Port))
			protocols = append(protocols, fmt.Sprint(*p.Protocol))
		}
		services := epSlice.Labels["kubernetes.io/service-name"]
		for _, endpoint := range epSlice.Endpoints {
			comDetails := &ComDetails{
				Direction:   "ingress",
				Protocol:    strings.Join(protocols, ","),
				Port:        strings.Join(ports, ","),
				NodeRole:    nodesRoles[*endpoint.NodeName],
				ServiceName: services,
				Required:    required,
			}
			res = append(res, *comDetails)
		}
	}
	res = RemoveDups(res)

	return ComMatrix{Matrix: res}, nil
}

func (m ComMatrix) ToCSV() ([]byte, error) {
	out := make([]byte, 0)
	w := bytes.NewBuffer(out)
	csvwriter := csv.NewWriter(w)

	for _, cd := range m.Matrix {
		record := strings.Split(cd.String(), ",")
		err := csvwriter.Write(record)
		if err != nil {
			return nil, fmt.Errorf("failed to convert to CSV foramt: %w", err)
		}
	}
	csvwriter.Flush()

	return w.Bytes(), nil
}

func RemoveDups(outPuts []ComDetails) []ComDetails {
	allKeys := make(map[string]bool)
	res := []ComDetails{}
	for _, item := range outPuts {
		str := fmt.Sprintf("%s-%s-%s", item.NodeRole, item.Port, item.Protocol)
		if _, value := allKeys[str]; !value {
			allKeys[str] = true
			res = append(res, item)
		}
	}

	return res
}

func GetNodesRoles(nodes *corev1.NodeList) map[string]string {
	res := make(map[string]string)

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

	return res
}
