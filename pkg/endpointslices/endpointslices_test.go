package endpointslices

import (
	"testing"
)

func TestNewQueryNilClient(t *testing.T) {
	_, err := NewQuery(nil)
	if err == nil {
		t.Fatalf("expected error for empty client")
	}
}
