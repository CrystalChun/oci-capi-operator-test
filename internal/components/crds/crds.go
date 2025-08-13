package crds

import (
	"context"
	"fmt"

	"github.com/openshift/oci-capi-operator/internal/utils"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	v1alpha3 "sigs.k8s.io/cluster-api/cmd/clusterctl/api/v1alpha3"
)

func GetComponents(ctx context.Context, provider string, providerType v1alpha3.ProviderType) ([]unstructured.Unstructured, error) {
	components, err := utils.GenerateCAPIComponents(ctx, provider, providerType, "")
	if err != nil {
		return nil, fmt.Errorf("error generating CAPI components: %w", err)
	}
	crds := []unstructured.Unstructured{}
	for _, component := range components {
		switch component.GetKind() {
		case "CustomResourceDefinition":
			utils.SetOpenshiftCABundleAnnotation(&component)
			crds = append(crds, component)
		}
	}
	return crds, nil
}
