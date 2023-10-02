package nodecommlib

import (
	"testing"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

func TestComm(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Coverage Test Suite")
}
