package commatrix

import (
	"bytes"
	"context"
	"encoding/csv"
	"fmt"
	"os"
	"strings"
	"text/tabwriter"

	corev1 "k8s.io/api/core/v1"
	discoveryv1 "k8s.io/api/discovery/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

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
	Required    string `json:"required"`
}

func (cd CommDetails) String() string {
	return fmt.Sprintf("%s,%s,%s,%s,%s,%s", cd.Direction, cd.Protocol, cd.Port, cd.NodeRole, cd.ServiceName, cd.Required)
}

func (m CommMatrix) PrintMat() {
	w := new(tabwriter.Writer)
	w.Init(os.Stdout, 12, 12, 0, '\t', 0)
	defer w.Flush()
	fmt.Fprintf(w, " %s\t\t\t%s\t\t\t%s\t\t\t%s\t\t\t%s\t\t\t%s\n", "DIRECTION", "NODE-ROLE", "PROTOCOL", "PORT", "SERVICE", "REQUIRED")

	for i, cd := range m.Matrix {
		fmt.Fprintf(w, "%d. %s\n", i+1, cd)
	}
}

func CreateCommMatrix(cs *client.ClientSet, slices []discoveryv1.EndpointSlice) (CommMatrix, error) {
	if len(slices) == 0 {
		return CommMatrix{}, fmt.Errorf("slices is empty")
	}

	nodes, err := cs.Nodes().List(context.TODO(), metav1.ListOptions{})
	if err != nil {
		return CommMatrix{}, err
	}

	nodesRoles := GetNodesRoles(nodes)
	res := make([]CommDetails, 0)

	for _, slice := range slices {
		required := "true"
		if _, ok := slice.Labels["optional"]; ok {
			required = "false"
		}
		ports := make([]string, 0)
		protocols := make([]string, 0)
		for _, p := range slice.Ports {
			ports = append(ports, fmt.Sprint(*p.Port))
			protocols = append(protocols, fmt.Sprint(*p.Protocol))
		}
		services := slice.Labels["kubernetes.io/service-name"]
		for _, endpoint := range slice.Endpoints {
			commDetails := &CommDetails{
				Direction:   "ingress",
				Protocol:    strings.Join(protocols, ","),
				Port:        strings.Join(ports, ","),
				NodeRole:    nodesRoles[*endpoint.NodeName],
				ServiceName: services,
				Required:    required,
			}
			res = append(res, *commDetails)
		}
	}
	res = RemoveDups(res)

	return CommMatrix{Matrix: res}, nil
}

func (m CommMatrix) ToCSV() ([]byte, error) {
	out := make([]byte, 0)
	w := bytes.NewBuffer(out)
	csvwriter := csv.NewWriter(w)

	for _, cd := range m.Matrix {
		record := strings.Split(cd.String(), ",")
		err := csvwriter.Write(record)
		if err != nil {
			return nil, err
		}
	}
	csvwriter.Flush()

	return w.Bytes(), nil
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
