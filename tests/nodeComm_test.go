package nodecommlib

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"strconv"
	"strings"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	corev1 "k8s.io/api/core/v1"
	discoveryv1 "k8s.io/api/discovery/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/liornoy/main/node-comm-lib/pkg/client"
	"github.com/liornoy/main/node-comm-lib/pkg/commatrix"
	"github.com/liornoy/main/node-comm-lib/pkg/endpointslices"
	"github.com/liornoy/main/node-comm-lib/pkg/pointer"
)

var (
	cs  *client.ClientSet
	err error
)

var _ = Describe("Comm Matrix", func() {
	BeforeEach(func() {
		cs, err = client.New("")
		Expect(err).ToNot(HaveOccurred())

		By("generating custom EndpointSlices for host services")
		err = createHostServiceSlices(cs)
		Expect(err).ToNot(HaveOccurred())

		By("fetching all ports cluster is listening to")
		_, err = exec.Command("./runSSonNodes.sh").Output()
		Expect(err).ToNot(HaveOccurred())
	})

	AfterEach(func() {
		By("fetching all custom EndpointSlices and deleting them")
		customeSlices, err := cs.EndpointSlices("default").List(context.TODO(), metav1.ListOptions{})
		Expect(err).ToNot(HaveOccurred())

		for _, slice := range customeSlices.Items {
			if !strings.Contains(slice.Name, "test") {
				continue
			}
			err := cs.EndpointSlices("default").Delete(context.TODO(), slice.Name, metav1.DeleteOptions{})
			Expect(err).ToNot(HaveOccurred())
		}
	})

	Context("create a comm matrix from the cluster", func() {
		It("should equal to what the nodes are actually listening on", func() {
			clusterComMat, err := generateClusterComMatrix(cs)
			Expect(err).ToNot(HaveOccurred())

			outfile, err := os.Create("./artifacts/ss-command-com-matrix.txt")
			Expect(err).ToNot(HaveOccurred())
			stdout := os.Stdout
			os.Stdout = outfile
			clusterComMat.PrintMat()
			outfile.Close()

			slices, err := endpointslices.GetIngressCommSlices(cs)
			Expect(err).ToNot(HaveOccurred())

			endpointSliceMat, err := commatrix.CreateCommMatrix(cs, slices)
			Expect(err).ToNot(HaveOccurred())

			outfile, err = os.Create("./artifacts/endpointslices-com-matirx.txt")
			Expect(err).ToNot(HaveOccurred())
			os.Stdout = outfile
			endpointSliceMat.PrintMat()
			outfile.Close()

			os.Stdout = stdout
		})
	})
})

func createHostServiceSlices(cs *client.ClientSet) error {
	nodes, err := cs.Nodes().List(context.TODO(), metav1.ListOptions{})
	if err != nil {
		return err
	}

	nodesRoles := commatrix.GetNodesRoles(nodes)
	rolesNodes := reverseMap(nodesRoles)

	slices, err := customHostServicesDefinition()
	if err != nil {
		return err
	}

	for _, s := range slices {
		port, err := strconv.ParseInt(s.Port, 10, 32)
		if err != nil {
			return err
		}
		name := fmt.Sprintf("test-%s-%s-%s", s.ServiceName, s.NodeRole, s.Port)
		name = strings.ToLower(name)

		nodeName := rolesNodes[s.NodeRole]

		endpointSlice := discoveryv1.EndpointSlice{
			ObjectMeta: metav1.ObjectMeta{
				Name:      name,
				Namespace: "default",
				Labels: map[string]string{"ingress": "",
					"kubernetes.io/service-name":             s.ServiceName,
					"endpointslice.kubernetes.io/managed-by": "com-matrix-operator",
				},
			},
			Ports: []discoveryv1.EndpointPort{
				{
					Port:     pointer.Int32Ptr(int32(port)),
					Protocol: (*corev1.Protocol)(&s.Protocol),
				},
			},
			Endpoints: []discoveryv1.Endpoint{
				{
					NodeName:  pointer.StrPtr(nodeName),
					Addresses: []string{"1.1.1.1"},
				},
			},
			AddressType: "IPv4",
		}
		if s.Required == "false" {
			endpointSlice.Labels["optional"] = "true"
		}

		_, err = cs.EndpointSlices("default").Create(context.TODO(), &endpointSlice, metav1.CreateOptions{})
		if err != nil && !errors.IsAlreadyExists(err) {
			return err
		}
	}

	return nil
}

func reverseMap(m map[string]string) map[string]string {
	n := make(map[string]string, len(m))
	for k, v := range m {
		n[v] = k
	}
	return n
}

func customHostServicesDefinition() ([]commatrix.CommDetails, error) {
	var res []commatrix.CommDetails
	bs, err := os.ReadFile("customEndpointSlices.json")
	if err != nil {
		return nil, err
	}

	err = json.Unmarshal(bs, &res)
	if err != nil {
		return nil, err
	}

	return res, nil
}

func generateClusterComMatrix(cs *client.ClientSet) (commatrix.CommMatrix, error) {
	var res = commatrix.CommMatrix{}

	nodes, err := cs.Nodes().List(context.TODO(), metav1.ListOptions{})
	Expect(err).ToNot(HaveOccurred())

	nodesRoles := commatrix.GetNodesRoles(nodes)

	comDetails := make([]commatrix.CommDetails, 0)
	for _, n := range nodes.Items {
		tcpOutput, err := os.ReadFile("./artifacts/" + n.Name + "-tcp.txt")
		Expect(err).ToNot(HaveOccurred())

		tcpComDetails := ssToCommDetails(string(tcpOutput), nodesRoles[n.Name], "TCP")
		comDetails = append(comDetails, tcpComDetails...)

		udpOutput, err := os.ReadFile("./artifacts/" + n.Name + "-udp.txt")
		Expect(err).ToNot(HaveOccurred())

		udpComDetails := ssToCommDetails(string(udpOutput), nodesRoles[n.Name], "UDP")
		comDetails = append(comDetails, udpComDetails...)
	}

	comDetails = commatrix.RemoveDups(comDetails)
	res.Matrix = comDetails

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
