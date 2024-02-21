package commatrix

import (
	"fmt"

	"github.com/liornoy/node-comm-lib/internal/client"
	"github.com/liornoy/node-comm-lib/internal/customendpointslices"
	"github.com/liornoy/node-comm-lib/internal/endpointslices"
	"github.com/liornoy/node-comm-lib/internal/types"
)

// New gets the kubeconfig path or consumes the KUBECONFIG env var
// and creates Communication Matrix for given cluster.
func New(kubeconfigPath string) (*types.ComMatrix, error) {
	cs, err := client.New(kubeconfigPath)
	if err != nil {
		return nil, fmt.Errorf("failed creating the client: %w", err)
	}

	// Temporary step: Manually creating missing endpointslices.
	err = customendpointslices.Create(cs)
	if err != nil {
		return nil, fmt.Errorf("failed creating custom services: %w", err)
	}

	epSlicesInfo, err := endpointslices.GetIngressEndpointSlices(cs)
	if err != nil {
		return nil, fmt.Errorf("failed getting endpointslices: %w", err)
	}

	comDetailsFromEndpointSlices, err := endpointslices.ToComDetails(cs, epSlicesInfo)
	if err != nil {
		return nil, err
	}

	res := &types.ComMatrix{Matrix: comDetailsFromEndpointSlices}

	return res, nil
}
