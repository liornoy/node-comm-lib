package main

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"text/tabwriter"

	discoveryv1 "k8s.io/api/discovery/v1"

	"github.com/liornoy/main/node-comm-lib/pkg/client"
	"github.com/liornoy/main/node-comm-lib/pkg/commatrix"
	"github.com/liornoy/main/node-comm-lib/pkg/endpointslices"
)

func main() {
	cs, err := client.New("")
	if err != nil {
		panic(err)
	}

	slices, err := endpointslices.GetIngressCommSlices(cs)
	if err != nil {
		panic(err)
	}

	if len(slices) == 0 {
		fmt.Println("GetIngressCommSlices returned no slices!")
		return
	}
	mat, err := commatrix.CreateCommMatrix(cs, slices)
	if err != nil {
		panic(err)
	}
	// printSlices(slices)
	printCommMat(mat)
}
func printCommMat(mat commatrix.CommMatrix) {
	b, err := json.Marshal(mat)
	if err != nil {
		fmt.Println(err)
		return
	}
	fmt.Println(string(b))
}

func printSlices(slices []discoveryv1.EndpointSlice) {
	w := new(tabwriter.Writer)
	w.Init(os.Stdout, 8, 8, 0, '\t', 0)
	defer w.Flush()

	fmt.Println("The returned slices from GetIngressCommSlices are:")
	fmt.Fprintf(w, " %s\t\t%s\t\t%s\n", "NAMESPACE", "NAME", "PORTS")
	fmt.Fprintf(w, " %s\t\t%s\t\t%s\n", "----", "----", "----")

	for i, slice := range slices {
		s := sliceToStr(slice)
		fmt.Fprintf(w, "%d. %s\n", i+1, s)
	}
}

func portsToString(endpointPorts []discoveryv1.EndpointPort) string {
	res := make([]string, 0)
	for _, endpoint := range endpointPorts {
		res = append(res, fmt.Sprint(*endpoint.Port))
	}

	return strings.Join(res, ",")
}

func sliceToStr(slice discoveryv1.EndpointSlice) string {
	if len(slice.OwnerReferences) == 0 || len(slice.Ports) == 0 {
		return ""
	}

	ports := make([]string, 0)
	for _, port := range slice.Ports {
		ports = append(ports, fmt.Sprint(*port.Port))
	}

	return fmt.Sprintf("%s\t\t%s\t\t%s", slice.Namespace, slice.Name, strings.Join(ports, ","))
}
