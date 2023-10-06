package endpointslices

import (
	"fmt"
	"testing"

	discoveryv1 "k8s.io/api/discovery/v1"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestNewQueryNilClient(t *testing.T) {
	_, err := NewQuery(nil)
	if err == nil {
		t.Fatalf("expected error for empty client")
	}
}

func TestWithLabels(t *testing.T) {
	var (
		noLabels      = map[string]string{}
		oneLabel      = map[string]string{"ingerss": ""}
		twoLabels     = map[string]string{"ingerss": "", "optional": "true"}
		nonexistLabel = map[string]string{"nonexist": ""}
		epSlices      = []discoveryv1.EndpointSlice{
			{
				ObjectMeta: v1.ObjectMeta{
					Name:   "epSliceNoLabels",
					Labels: noLabels,
				},
			},
			{
				ObjectMeta: v1.ObjectMeta{
					Name:   "epSliceOneLabel",
					Labels: oneLabel,
				},
			},
			{
				ObjectMeta: v1.ObjectMeta{
					Name:   "epSliceTwoLabels",
					Labels: twoLabels,
				},
			},
		}
		queryParams = QueryParams{
			epSlices: epSlices,
			filter:   make([]bool, len(epSlices)),
		}
	)

	tests := []struct {
		q               QueryParams
		desc            string
		labels          map[string]string
		expectedEpSlice map[string]bool
	}{
		{
			q:      queryParams,
			desc:   "with-no-labels",
			labels: noLabels,
			expectedEpSlice: map[string]bool{
				"epSliceNoLabels":  true,
				"epSliceOneLabel":  true,
				"epSliceTwoLabels": true,
			},
		},
		{
			q:      queryParams,
			desc:   "with-one-label",
			labels: oneLabel,
			expectedEpSlice: map[string]bool{
				"epSliceOneLabel":  true,
				"epSliceTwoLabels": true,
			},
		},
		{
			q:      queryParams,
			desc:   "with-two-labels",
			labels: twoLabels,
			expectedEpSlice: map[string]bool{
				"epSliceTwoLabels": true,
			},
		},
		{
			q:               queryParams,
			desc:            "with-nonexist-label",
			labels:          nonexistLabel,
			expectedEpSlice: map[string]bool{},
		},
	}
	for _, test := range tests {
		resetFilter(&test.q)
		res := test.q.WithLabels(test.labels).Query()
		if err := isEqual(res, test.expectedEpSlice); err != nil {
			t.Fatalf("test \"%s\" failed: %s", test.desc, err)
		}
	}
}

func isEqual(epSlices []discoveryv1.EndpointSlice, expected map[string]bool) error {
	if len(epSlices) != len(expected) {
		return fmt.Errorf("got %d epSlices, expected %d", len(epSlices), len(expected))
	}

	for _, epSlice := range epSlices {
		if _, ok := expected[epSlice.Name]; !ok {
			return fmt.Errorf("got unexpected epSlice \"%s\"", epSlice.Name)
		}
	}

	return nil
}

func resetFilter(q *QueryParams) {
	for i := range q.filter {
		q.filter[i] = false
	}
}
