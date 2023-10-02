package nodecommlib

import (
	"bufio"
	"context"
	"fmt"
	"os"
	"strings"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	_ "github.com/openshift/cluster-node-tuning-operator/test/e2e/performanceprofile/functests/utils/client" // This is to init the nto client
	"github.com/openshift/cluster-node-tuning-operator/test/e2e/performanceprofile/functests/utils/nodes"
	discoveryv1 "k8s.io/api/discovery/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/liornoy/main/node-comm-lib/pkg/client"
	"github.com/liornoy/main/node-comm-lib/pkg/commatrix"
	"github.com/liornoy/main/node-comm-lib/pkg/endpointslices"
)

type TheOuput struct {
	State        string `json:"State"`
	RecvQ        string `json:"Recv-Q"`
	SendQ        string `json:"Send-Q"`
	AddrPort     string `json:"Local Address:Port"`
	PeerAddrPort string `json:"Peer Address:Port"`
	Process      string `json:"Process"`
}

var _ = Describe("Comm Matrix", func() {
	Context("create a comm matrix from the cluster", func() {
		It("should equal to what the nodes are actually listening on", func() {
			stdout := os.Stdout

			cs, err := client.New("")
			Expect(err).ToNot(HaveOccurred())

			clusterComMat, err := generateClusterComMatrix(cs)
			Expect(err).ToNot(HaveOccurred())

			outfile, err := os.Create("./artifacts/clusterComMat.txt")
			Expect(err).ToNot(HaveOccurred())
			os.Stdout = outfile
			clusterComMat.PrintMat()
			outfile.Close()

			slices, err := endpointslices.GetIngressCommSlices(cs)
			Expect(err).ToNot(HaveOccurred())

			if len(slices) == 0 {
				fmt.Println("GetIngressCommSlices returned no slices!")
				return
			}
			endpointSliceMat, err := commatrix.CreateCommMatrix(cs, slices)
			Expect(err).ToNot(HaveOccurred())

			outfile, err = os.Create("./artifacts/endpointSlicesComMan.txt")
			Expect(err).ToNot(HaveOccurred())
			os.Stdout = outfile
			endpointSliceMat.PrintMat()
			outfile.Close()

			os.Stdout = stdout
		})
	})
})

func generateClusterComMatrix(cs *client.ClientSet) (commatrix.CommMatrix, error) {
	var res = commatrix.CommMatrix{}

	// Get open ports from nodes and create Comm Matrix to compare
	dnodes, err := cs.Nodes().List(context.TODO(), metav1.ListOptions{})
	Expect(err).ToNot(HaveOccurred())

	nodesRoles, err := commatrix.GetNodesRoles(cs)
	Expect(err).ToNot(HaveOccurred())

	outPuts := make([]commatrix.CommDetails, 0)
	for _, n := range dnodes.Items {
		tcpOutput, err := nodes.ExecCommandOnNode([]string{"ss", "-plant"}, &n)
		Expect(err).ToNot(HaveOccurred())

		//Debug
		outfile, err := os.Create("./artifacts/" + n.Name + "-tcp.txt")
		Expect(err).ToNot(HaveOccurred())
		outfile.WriteString(tcpOutput)
		outfile.Close()
		//

		tcpComDetails := ssToCommDetails(tcpOutput, nodesRoles[n.Name], "TCP")
		outPuts = append(outPuts, tcpComDetails...)

		udpOutput, err := nodes.ExecCommandOnNode([]string{"ss", "-planu"}, &n)
		Expect(err).ToNot(HaveOccurred())

		//Debug
		outfile, err = os.Create("./artifacts/" + n.Name + "-udp.txt")
		Expect(err).ToNot(HaveOccurred())
		outfile.WriteString(udpOutput)
		outfile.Close()
		//

		udpComDetails := ssToCommDetails(udpOutput, nodesRoles[n.Name], "UDP")
		outPuts = append(outPuts, udpComDetails...)
	}

	outPuts = commatrix.RemoveDups(outPuts)
	res.Matrix = outPuts

	return res, nil
}

func ssToCommDetails(ssOutput string, role string, protocol string) []commatrix.CommDetails {
	res := make([]commatrix.CommDetails, 0)

	reader := strings.NewReader(ssOutput)
	scanner := bufio.NewScanner(reader)

	for scanner.Scan() {
		text := scanner.Text()

		if strings.Contains(text, "127.0.0") {
			continue
		}
		if protocol == "TCP" && !strings.Contains(text, "LISTEN") {
			continue
		}
		if protocol == "UDP" && !strings.Contains(text, "ESTAB") {
			continue
		}
		tokens := strings.Fields(text)
		if len(tokens) < 4 {
			continue
		}

		process := "empty"
		if len(tokens) == 6 {
			process = getInDoubleQuotes(tokens[5])
		}

		idx := strings.LastIndex(tokens[3], ":")
		port := tokens[3][idx+1:]

		res = append(res, commatrix.CommDetails{
			Direction:   "ingress",
			Protocol:    protocol,
			Port:        port,
			NodeRole:    role,
			ServiceName: process})
	}

	return res
}

func portsToString(endpointPorts []discoveryv1.EndpointPort) string {
	res := make([]string, 0)
	for _, endpoint := range endpointPorts {
		res = append(res, fmt.Sprint(*endpoint.Port))
	}

	return strings.Join(res, ",")
}

func getInDoubleQuotes(s string) string {
	res := make([]string, 0)
	for idx, endIdx := 0, 0; strings.Index(s, "\"") != -1; s = s[idx+endIdx+2:] {
		idx = strings.Index(s, "\"")
		endIdx = strings.Index(s[idx+1:], "\"")
		res = append(res, s[idx+1:idx+1+endIdx])
	}
	return strings.Join(res, ",")
}
