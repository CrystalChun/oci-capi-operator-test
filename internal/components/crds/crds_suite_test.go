package crds

import (
	"testing"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

func TestCRDs(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "CRDs Suite")
}