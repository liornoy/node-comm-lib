package endpointslices

import (
	"fmt"
	"testing"

	corev1 "k8s.io/api/core/v1"
	discoveryv1 "k8s.io/api/discovery/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/liornoy/main/node-comm-lib/pkg/consts"
	"github.com/liornoy/main/node-comm-lib/pkg/fakeclient"
)

func TestNewQueryNilClient(t *testing.T) {
	_, err := NewQuery(nil)
	if err == nil {
		t.Fatalf("expected error for empty client")
	}
}

func TestNewQuery(t *testing.T) {
	var (
		initObjects = fakeclient.ClusterResources{
			Pods: []corev1.Pod{
				{
					ObjectMeta: metav1.ObjectMeta{
						Name: "pod1",
					},
				},
				{
					ObjectMeta: metav1.ObjectMeta{
						Name: "pod2",
					},
				},
			},
			EpSlices: []discoveryv1.EndpointSlice{
				{
					ObjectMeta: metav1.ObjectMeta{
						Name: "epslice1",
					},
				},
				{
					ObjectMeta: metav1.ObjectMeta{
						Name: "epslice2",
					},
				},
			},
			Services: []corev1.Service{
				{
					ObjectMeta: metav1.ObjectMeta{
						Name: "service1",
					},
				},
				{
					ObjectMeta: metav1.ObjectMeta{
						Name: "serivce2",
					},
				},
			},
		}
	)

	c, err := fakeclient.New(fakeclient.ObjectsFromResources(initObjects))
	if err != nil {
		t.Fatalf("failed to create client: %s", err)
	}

	q, err := NewQuery(c)
	if err != nil {
		t.Fatalf("failed to create new query: %s", err)
	}

	if len(q.pods) != len(initObjects.Pods) {
		t.Fatalf("queryParam has %d Pods, expected %d", len(q.pods), len(initObjects.Pods))
	}

	if len(q.epSlices) != len(initObjects.EpSlices) {
		t.Fatalf("queryParam has %d EndpointSlices, expected %d", len(q.epSlices), len(initObjects.EpSlices))
	}

	if len(q.services) != len(initObjects.Services) {
		t.Fatalf("queryParam has %d Services, expected %d", len(q.services), len(initObjects.Services))
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
				ObjectMeta: metav1.ObjectMeta{
					Name:   "epslice-no-labels",
					Labels: noLabels,
				},
			},
			{
				ObjectMeta: metav1.ObjectMeta{
					Name:   "epslice-one-label",
					Labels: oneLabel,
				},
			},
			{
				ObjectMeta: metav1.ObjectMeta{
					Name:   "epslice-two-labels",
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
				"epslice-no-labels":  true,
				"epslice-one-label":  true,
				"epslice-two-labels": true,
			},
		},
		{
			q:      queryParams,
			desc:   "with-one-label",
			labels: oneLabel,
			expectedEpSlice: map[string]bool{
				"epslice-one-label":  true,
				"epslice-two-labels": true,
			},
		},
		{
			q:      queryParams,
			desc:   "with-two-labels",
			labels: twoLabels,
			expectedEpSlice: map[string]bool{
				"epslice-two-labels": true,
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
				ObjectMeta: metav1.ObjectMeta{
					Name: "epslice1",
				},
			},
			{
				ObjectMeta: metav1.ObjectMeta{
					Name: "epslice2",
				},
			},
			{
				ObjectMeta: metav1.ObjectMeta{
					Name: "epslice3",
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
				"epslice1": true,
				"epslice2": true,
				"epslice3": true,
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
			expectedEpSlice: map[string]bool{"epslice1": true},
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

func TestWithHostNetwork(t *testing.T) {
	var (
		pods = []corev1.Pod{
			{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "hostnetwork-pod",
					Namespace: consts.TestNameSpace,
				},
				Spec: corev1.PodSpec{
					HostNetwork: true,
				},
			},
			{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "non-hostnetwork-pod",
					Namespace: consts.TestNameSpace,
				},
				Spec: corev1.PodSpec{
					HostNetwork: false,
				},
			},
		}
		epSlices = []discoveryv1.EndpointSlice{
			{
				ObjectMeta: metav1.ObjectMeta{
					Name: "with-hostnetwork",
				},
				Endpoints: []discoveryv1.Endpoint{
					{
						TargetRef: &corev1.ObjectReference{
							Name:      "hostnetwork-pod",
							Namespace: consts.TestNameSpace,
						},
					},
				},
			},
			{
				ObjectMeta: metav1.ObjectMeta{
					Name: "with-non-hostnetwork",
				},
				Endpoints: []discoveryv1.Endpoint{
					{
						TargetRef: &corev1.ObjectReference{
							Name:      "non-hostnetwork-pod",
							Namespace: consts.TestNameSpace,
						},
					},
				},
			},
			{
				ObjectMeta: metav1.ObjectMeta{
					Name: "with-no-endpoints",
				},
			},
			{
				ObjectMeta: metav1.ObjectMeta{
					Name: "with-no-target-ref",
				},
				Endpoints: []discoveryv1.Endpoint{
					{
						Addresses: []string{"1.1.1.1"},
					},
				},
			},
		}
		q = QueryParams{
			epSlices: epSlices,
			pods:     pods,
			filter:   make([]bool, len(epSlices)),
		}
	)

	expectedEpSlice := map[string]bool{
		"with-hostnetwork": true,
	}

	res := q.WithHostNetwork().Query()
	if err := isEqual(res, expectedEpSlice); err != nil {
		t.Fatalf("test \"with-hostnetwork\" failed: %s", err)
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
