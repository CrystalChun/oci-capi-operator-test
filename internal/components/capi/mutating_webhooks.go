package capi

import (
	capiv1alpha1 "github.com/openshift/oci-capi-operator/api/v1alpha1"
	admissionregistrationv1 "k8s.io/api/admissionregistration/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
)

// MutatingWebhookConfiguration creates the MutatingWebhookConfiguration for CAPI
func MutatingWebhookConfiguration(capiSystemNamespace string, autoscaler *capiv1alpha1.OCIClusterAutoscaler, scheme *runtime.Scheme) (client.Object, func() error) {
	webhook := &admissionregistrationv1.MutatingWebhookConfiguration{
		ObjectMeta: metav1.ObjectMeta{
			Name: "capi-mutating-webhook-configuration",
			Annotations: map[string]string{
				OCPCertAnnotationKey: OCPCertAnnotationValue,
			},
			Labels: map[string]string{
				capiProviderLabel:   capiProviderValue,
				capiClusterctlLabel: "",
			},
		},
		Webhooks: buildMutatingWebhooks(capiSystemNamespace),
	}

	mutateFn := func() error {
		return controllerutil.SetControllerReference(autoscaler, webhook, scheme)
	}

	return webhook, mutateFn
}

func buildMutatingWebhooks(namespace string) []admissionregistrationv1.MutatingWebhook {
	commonConfig := defaultCommonConfig()
	createUpdateOps := []admissionregistrationv1.OperationType{admissionregistrationv1.Create, admissionregistrationv1.Update}

	webhooks := []struct {
		builder WebhookBuilder
		ops     []admissionregistrationv1.OperationType
	}{
		{
			builder: WebhookBuilder{
				Name:     "default.cluster.cluster.x-k8s.io",
				Path:     "/mutate-cluster-x-k8s-io-v1beta1-cluster",
				APIGroup: "cluster.x-k8s.io",
				Version:  "v1beta1",
				Resource: "clusters",
			},
			ops: createUpdateOps,
		},
		{
			builder: WebhookBuilder{
				Name:     "default.clusterclass.cluster.x-k8s.io",
				Path:     "/mutate-cluster-x-k8s-io-v1beta1-clusterclass",
				APIGroup: "cluster.x-k8s.io",
				Version:  "v1beta1",
				Resource: "clusterclasses",
			},
			ops: createUpdateOps,
		},
		{
			builder: WebhookBuilder{
				Name:     "default.clusterresourceset.addons.cluster.x-k8s.io",
				Path:     "/mutate-addons-cluster-x-k8s-io-v1beta1-clusterresourceset",
				APIGroup: "addons.cluster.x-k8s.io",
				Version:  "v1beta1",
				Resource: "clusterresourcesets",
			},
			ops: createUpdateOps,
		},
		{
			builder: WebhookBuilder{
				Name:     "default.machine.cluster.x-k8s.io",
				Path:     "/mutate-cluster-x-k8s-io-v1beta1-machine",
				APIGroup: "cluster.x-k8s.io",
				Version:  "v1beta1",
				Resource: "machines",
			},
			ops: createUpdateOps,
		},
		{
			builder: WebhookBuilder{
				Name:     "default.machinedeployment.cluster.x-k8s.io",
				Path:     "/mutate-cluster-x-k8s-io-v1beta1-machinedeployment",
				APIGroup: "cluster.x-k8s.io",
				Version:  "v1beta1",
				Resource: "machinedeployments",
			},
			ops: createUpdateOps,
		},
		{
			builder: WebhookBuilder{
				Name:     "default.machinehealthcheck.cluster.x-k8s.io",
				Path:     "/mutate-cluster-x-k8s-io-v1beta1-machinehealthcheck",
				APIGroup: "cluster.x-k8s.io",
				Version:  "v1beta1",
				Resource: "machinehealthchecks",
			},
			ops: createUpdateOps,
		},
		{
			builder: WebhookBuilder{
				Name:     "default.machineset.cluster.x-k8s.io",
				Path:     "/mutate-cluster-x-k8s-io-v1beta1-machineset",
				APIGroup: "cluster.x-k8s.io",
				Version:  "v1beta1",
				Resource: "machinesets",
			},
			ops: createUpdateOps,
		},
		{
			builder: WebhookBuilder{
				Name:     "default.extensionconfig.runtime.addons.cluster.x-k8s.io",
				Path:     "/mutate-runtime-cluster-x-k8s-io-v1alpha1-extensionconfig",
				APIGroup: "runtime.cluster.x-k8s.io",
				Version:  "v1alpha1",
				Resource: "extensionconfigs",
			},
			ops: createUpdateOps,
		},
		{
			builder: WebhookBuilder{
				Name:     "default.machinepool.cluster.x-k8s.io",
				Path:     "/mutate-cluster-x-k8s-io-v1beta1-machinepool",
				APIGroup: "cluster.x-k8s.io",
				Version:  "v1beta1",
				Resource: "machinepools",
			},
			ops: createUpdateOps,
		},
	}

	var mutatingWebhooks []admissionregistrationv1.MutatingWebhook
	for _, w := range webhooks {
		webhook := admissionregistrationv1.MutatingWebhook{
			Name: w.builder.Name,
			ClientConfig: admissionregistrationv1.WebhookClientConfig{
				Service: serviceReference(w.builder.Path, namespace),
			},
			Rules:                   buildWebhookRules(w.builder.APIGroup, w.builder.Version, w.builder.Resource, w.ops),
			AdmissionReviewVersions: commonConfig.AdmissionReviewVersions,
			SideEffects:             commonConfig.SideEffects,
			TimeoutSeconds:          commonConfig.TimeoutSeconds,
			FailurePolicy:           commonConfig.FailurePolicy,
			MatchPolicy:             commonConfig.MatchPolicy,
			ReinvocationPolicy:      reinvocationPolicyPtr(admissionregistrationv1.NeverReinvocationPolicy),
		}
		mutatingWebhooks = append(mutatingWebhooks, webhook)
	}

	return mutatingWebhooks
}
