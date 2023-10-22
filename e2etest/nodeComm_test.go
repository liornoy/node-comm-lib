package e2etest

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path"
	"reflect"
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
	"github.com/liornoy/main/node-comm-lib/pkg/consts"
	"github.com/liornoy/main/node-comm-lib/pkg/endpointslices"
	"github.com/liornoy/main/node-comm-lib/pkg/pointer"
	"github.com/liornoy/main/node-comm-lib/pkg/ss"
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
		_, err = exec.Command("./hack/runSSonNodes.sh").Output()
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
			expectedComMat, err := generateClusterComMatrix(cs)
			Expect(err).ToNot(HaveOccurred())

			epSliceQuery, err := endpointslices.NewQuery(cs)
			Expect(err).ToNot(HaveOccurred())

			ingressSlice := epSliceQuery.
				WithHostNetwork().
				WithLabels(map[string]string{consts.IngressLabel: ""}).
				WithServiceType(corev1.ServiceTypeNodePort).
				WithServiceType(corev1.ServiceTypeLoadBalancer).
				Query()

			endpointSliceMat, err := commatrix.CreateComMatrix(cs, ingressSlice)
			Expect(err).ToNot(HaveOccurred())

			err = printArtifacts(expectedComMat, endpointSliceMat)
			Expect(err).ToNot(HaveOccurred())

			printMatDiff(endpointSliceMat, expectedComMat)

			Expect(reflect.DeepEqual(endpointSliceMat, expectedComMat)).To(BeTrue(),
				"expected communication matrix different than generated")
		})
	})
})

func calcMatDiff(m1 commatrix.ComMatrix, m2 commatrix.ComMatrix) []commatrix.ComDetails {
	diff := []commatrix.ComDetails{}
	for _, cd1 := range m1.Matrix {
		found := false
		for _, cd2 := range m2.Matrix {
			if cd1.Port == cd2.Port {
				found = true
				break
			}
		}
		if !found {
			diff = append(diff, cd1)
		}
	}

	return diff
}

func printMatDiff(m1 commatrix.ComMatrix, m2 commatrix.ComMatrix) {
	diffMat1 := calcMatDiff(m1, m2)
	diffMat2 := calcMatDiff(m2, m1)

	if len(diffMat1) == 0 && len(diffMat2) == 0 {
		fmt.Println("matrices are equal")
		return
	}

	fmt.Println("In matrix1 but not in matrix2:")
	for _, cd := range diffMat1 {
		fmt.Printf("%s - %s\n", cd.Port, cd.ServiceName)
	}

	fmt.Println("\nIn matrix2 but not in matrix1:")
	for _, cd := range diffMat2 {
		fmt.Printf("%s - %s\n", cd.Port, cd.ServiceName)
	}
}

func printArtifacts(ssComMat commatrix.ComMatrix, slicesComMat commatrix.ComMatrix) error {
	ssComMatPath := path.Join(artifactsPath, "ss-command-com-matrix.txt")
	outfile, err := os.Create(ssComMatPath)
	if err != nil {
		return err
	}

	defer outfile.Close()
	err = ssComMat.WriteTo(outfile)
	if err != nil {
		return err
	}
	outfile.Close()

	slicesComMatPath := path.Join(artifactsPath, "endpointslices-com-matirx.txt")
	outfile, err = os.Create(slicesComMatPath)
	if err != nil {
		return err
	}
	defer outfile.Close()
	err = slicesComMat.WriteTo(outfile)
	if err != nil {
		return err
	}

	return nil
}

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
		if !s.Required {
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

func customHostServicesDefinition() ([]commatrix.ComDetails, error) {
	var res []commatrix.ComDetails
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

func generateClusterComMatrix(cs *client.ClientSet) (commatrix.ComMatrix, error) {
	var res = commatrix.ComMatrix{}

	nodes, err := cs.Nodes().List(context.TODO(), metav1.ListOptions{})
	Expect(err).ToNot(HaveOccurred())

	nodesRoles := commatrix.GetNodesRoles(nodes)

	comDetails := make([]commatrix.ComDetails, 0)
	for _, n := range nodes.Items {
		tcpFileName := n.Name + "-tcp.txt"
		tcpOutput, err := os.ReadFile(path.Join(artifactsPath, tcpFileName))
		Expect(err).ToNot(HaveOccurred())

		tcpComDetails := ss.ToComDetails(string(tcpOutput), nodesRoles[n.Name], "TCP")
		comDetails = append(comDetails, tcpComDetails...)

		udpFileName := n.Name + "-udp.txt"
		udpOutput, err := os.ReadFile(path.Join(artifactsPath, udpFileName))
		Expect(err).ToNot(HaveOccurred())

		udpComDetails := ss.ToComDetails(string(udpOutput), nodesRoles[n.Name], "UDP")
		comDetails = append(comDetails, udpComDetails...)
	}

	comDetails = commatrix.RemoveDups(comDetails)
	res.Matrix = comDetails

	return res, nil
}

func portsToString(endpointPorts []discoveryv1.EndpointPort) string {
	res := make([]string, 0)
	for _, endpoint := range endpointPorts {
		res = append(res, fmt.Sprint(*endpoint.Port))
	}

	return strings.Join(res, ",")
}
