package enableautoscaler

import (
	"testing"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

func TestEnableAutoscaler(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Enable Autoscaler Suite")
}