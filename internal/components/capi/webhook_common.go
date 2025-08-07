package capi

import (
	"github.com/go-openapi/swag"
	admissionregistrationv1 "k8s.io/api/admissionregistration/v1"
)

const (
	capiWebhookServiceName = "capi-webhook-service"
	capiWebhookServicePort = 443
	OCPCertAnnotationKey   = "service.beta.openshift.io/inject-cabundle"
	OCPCertAnnotationValue = "true"
	capiProviderLabel      = "cluster.x-k8s.io/provider"
	capiProviderValue      = "cluster-api"
	capiClusterctlLabel    = "clusterctl.cluster.x-k8s.io"
)

// WebhookBuilder contains common configuration for building webhooks
type WebhookBuilder struct {
	Name      string
	Namespace string
	Path      string
	APIGroup  string
	Resource  string
	Version   string
}

// CommonWebhookConfig contains shared webhook configuration
type CommonWebhookConfig struct {
	AdmissionReviewVersions []string
	SideEffects             *admissionregistrationv1.SideEffectClass
	TimeoutSeconds          *int32
	FailurePolicy           *admissionregistrationv1.FailurePolicyType
	MatchPolicy             *admissionregistrationv1.MatchPolicyType
}

func defaultCommonConfig() CommonWebhookConfig {
	return CommonWebhookConfig{
		AdmissionReviewVersions: []string{"v1", "v1beta1"},
		SideEffects:             sideEffectPtr(admissionregistrationv1.SideEffectClassNone),
		TimeoutSeconds:          swag.Int32(10),                                     // default is 10 seconds
		FailurePolicy:           failurePolicyPtr(admissionregistrationv1.Fail),     // default is Fail
		MatchPolicy:             matchPolicyPtr(admissionregistrationv1.Equivalent), // default is Equivalent
	}
}

func serviceReference(path, namespace string) *admissionregistrationv1.ServiceReference {
	return &admissionregistrationv1.ServiceReference{
		Name:      capiWebhookServiceName,
		Namespace: namespace,
		Path:      swag.String(path),
		Port:      swag.Int32(capiWebhookServicePort),
	}
}

func buildWebhookRules(apiGroup, version, resource string, operations []admissionregistrationv1.OperationType) []admissionregistrationv1.RuleWithOperations {
	return []admissionregistrationv1.RuleWithOperations{
		{
			Operations: operations,
			Rule: admissionregistrationv1.Rule{
				APIGroups:   []string{apiGroup},
				APIVersions: []string{version},
				Resources:   []string{resource},
				Scope:       scopePtr(admissionregistrationv1.AllScopes),
			},
		},
	}
}

// Helper functions for pointer types
func scopePtr(scope admissionregistrationv1.ScopeType) *admissionregistrationv1.ScopeType {
	return &scope
}

func sideEffectPtr(sideEffect admissionregistrationv1.SideEffectClass) *admissionregistrationv1.SideEffectClass {
	return &sideEffect
}

func failurePolicyPtr(policy admissionregistrationv1.FailurePolicyType) *admissionregistrationv1.FailurePolicyType {
	return &policy
}

func matchPolicyPtr(policy admissionregistrationv1.MatchPolicyType) *admissionregistrationv1.MatchPolicyType {
	return &policy
}

func reinvocationPolicyPtr(policy admissionregistrationv1.ReinvocationPolicyType) *admissionregistrationv1.ReinvocationPolicyType {
	return &policy
}
