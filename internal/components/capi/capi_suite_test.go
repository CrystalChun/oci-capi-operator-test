package capi

import (
	"testing"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

func TestCAPI(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "CAPI Suite")
}