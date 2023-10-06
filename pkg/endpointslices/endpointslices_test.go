package endpointslices

import (
	"fmt"
	"testing"

	discoveryv1 "k8s.io/api/discovery/v1"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/liornoy/main/node-comm-lib/pkg/consts"
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
		oneLabel      = map[string]string{consts.IngressLabel: ""}
		twoLabels     = map[string]string{consts.IngressLabel: "", "optional": "true"}
		mixedLabels   = map[string]string{consts.IngressLabel: "", "nonexist": ""}
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
			desc:            "with-exist-and-nonexist-labels",
			labels:          mixedLabels,
			expectedEpSlice: map[string]bool{},
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

func TestQuery(t *testing.T) {
	var (
		filterAll   = []bool{true, true, true}
		filterNone  = []bool{false, false, false}
		filterFirst = []bool{true, false, false}
		epSlices    = []discoveryv1.EndpointSlice{
			{
				ObjectMeta: v1.ObjectMeta{
					Name: "epSlice1",
				},
			},
			{
				ObjectMeta: v1.ObjectMeta{
					Name: "epSlice2",
				},
			},
			{
				ObjectMeta: v1.ObjectMeta{
					Name: "epSlice3",
				},
			},
		}
		queryParams = QueryParams{
			epSlices: epSlices,
		}
	)

	tests := []struct {
		q               QueryParams
		desc            string
		filter          []bool
		expectedEpSlice map[string]bool
	}{
		{
			q:      queryParams,
			desc:   "filter-all",
			filter: filterAll,
			expectedEpSlice: map[string]bool{
				"epSlice1": true,
				"epSlice2": true,
				"epSlice3": true,
			},
		},
		{
			q:               queryParams,
			desc:            "filter-none",
			filter:          filterNone,
			expectedEpSlice: map[string]bool{},
		},
		{
			q:               queryParams,
			desc:            "filter-first",
			filter:          filterFirst,
			expectedEpSlice: map[string]bool{"epSlice1": true},
		},
	}
	for _, test := range tests {
		test.q.filter = test.filter
		res := test.q.Query()
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
