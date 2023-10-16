package e2etest

import (
	"os"
	"path"
	"testing"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

const artifactsDir = "artifacts"

var artifactsPath string

func init() {
	currentPath, err := os.Getwd()
	if err != nil {
		panic(err)
	}
	artifactsPath = path.Join(currentPath, artifactsDir)
	os.Setenv("ARTIFACTS", artifactsPath)
}

func TestComm(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Coverage Test Suite")
}
