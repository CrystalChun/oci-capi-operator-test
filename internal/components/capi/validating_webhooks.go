package capi

import (
	capiv1alpha1 "github.com/openshift/oci-capi-operator/api/v1alpha1"
	admissionregistrationv1 "k8s.io/api/admissionregistration/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
)

// ValidatingWebhookConfiguration creates the ValidatingWebhookConfiguration for CAPI
func ValidatingWebhookConfiguration(capiSystemNamespace string, autoscaler *capiv1alpha1.OCIClusterAutoscaler, scheme *runtime.Scheme) (client.Object, func() error) {
	webhook := &admissionregistrationv1.ValidatingWebhookConfiguration{
		ObjectMeta: metav1.ObjectMeta{
			Name: "capi-validating-webhook-configuration",
			Annotations: map[string]string{
				OCPCertAnnotationKey: OCPCertAnnotationValue,
			},
			Labels: map[string]string{
				capiProviderLabel:   capiProviderValue,
				capiClusterctlLabel: "",
			},
		},
		Webhooks: buildValidatingWebhooks(capiSystemNamespace),
	}

	mutateFn := func() error {
		return controllerutil.SetControllerReference(autoscaler, webhook, scheme)
	}

	return webhook, mutateFn
}

func buildValidatingWebhooks(namespace string) []admissionregistrationv1.ValidatingWebhook {
	commonConfig := defaultCommonConfig()
	createUpdateOps := []admissionregistrationv1.OperationType{admissionregistrationv1.Create, admissionregistrationv1.Update}
	createUpdateDeleteOps := append(createUpdateOps, admissionregistrationv1.Delete)

	webhooks := []struct {
		builder WebhookBuilder
		ops     []admissionregistrationv1.OperationType
	}{
		{
			builder: WebhookBuilder{
				Name:     "validation.cluster.cluster.x-k8s.io",
				Path:     "/validate-cluster-x-k8s-io-v1beta1-cluster",
				APIGroup: "cluster.x-k8s.io",
				Version:  "v1beta1",
				Resource: "clusters",
			},
			ops: createUpdateDeleteOps,
		},
		{
			builder: WebhookBuilder{
				Name:     "validation.clusterclass.cluster.x-k8s.io",
				Path:     "/validate-cluster-x-k8s-io-v1beta1-clusterclass",
				APIGroup: "cluster.x-k8s.io",
				Version:  "v1beta1",
				Resource: "clusterclasses",
			},
			ops: createUpdateDeleteOps,
		},
		{
			builder: WebhookBuilder{
				Name:     "validation.clusterresourceset.addons.cluster.x-k8s.io",
				Path:     "/validate-addons-cluster-x-k8s-io-v1beta1-clusterresourceset",
				APIGroup: "addons.cluster.x-k8s.io",
				Version:  "v1beta1",
				Resource: "clusterresourcesets",
			},
			ops: createUpdateOps,
		},
		{
			builder: WebhookBuilder{
				Name:     "validation.clusterresourcesetbinding.addons.cluster.x-k8s.io",
				Path:     "/validate-addons-cluster-x-k8s-io-v1beta1-clusterresourcesetbinding",
				APIGroup: "addons.cluster.x-k8s.io",
				Version:  "v1beta1",
				Resource: "clusterresourcesetbindings",
			},
			ops: createUpdateOps,
		},
		{
			builder: WebhookBuilder{
				Name:     "validation.machine.cluster.x-k8s.io",
				Path:     "/validate-cluster-x-k8s-io-v1beta1-machine",
				APIGroup: "cluster.x-k8s.io",
				Version:  "v1beta1",
				Resource: "machines",
			},
			ops: createUpdateOps,
		},
		{
			builder: WebhookBuilder{
				Name:     "validation.machinedeployment.cluster.x-k8s.io",
				Path:     "/validate-cluster-x-k8s-io-v1beta1-machinedeployment",
				APIGroup: "cluster.x-k8s.io",
				Version:  "v1beta1",
				Resource: "machinedeployments",
			},
			ops: createUpdateOps,
		},
		{
			builder: WebhookBuilder{
				Name:     "validation.machinedrainrule.cluster.x-k8s.io",
				Path:     "/validate-cluster-x-k8s-io-v1beta1-machinedrainrule",
				APIGroup: "cluster.x-k8s.io",
				Version:  "v1beta1",
				Resource: "machinedrainrules",
			},
			ops: createUpdateOps,
		},
		{
			builder: WebhookBuilder{
				Name:     "validation.machinehealthcheck.cluster.x-k8s.io",
				Path:     "/validate-cluster-x-k8s-io-v1beta1-machinehealthcheck",
				APIGroup: "cluster.x-k8s.io",
				Version:  "v1beta1",
				Resource: "machinehealthchecks",
			},
			ops: createUpdateOps,
		},
		{
			builder: WebhookBuilder{
				Name:     "validation.machineset.cluster.x-k8s.io",
				Path:     "/validate-cluster-x-k8s-io-v1beta1-machineset",
				APIGroup: "cluster.x-k8s.io",
				Version:  "v1beta1",
				Resource: "machinesets",
			},
			ops: createUpdateOps,
		},
		{
			builder: WebhookBuilder{
				Name:     "validation.extensionconfig.runtime.cluster.x-k8s.io",
				Path:     "/validate-runtime-cluster-x-k8s-io-v1alpha1-extensionconfig",
				APIGroup: "runtime.cluster.x-k8s.io",
				Version:  "v1alpha1",
				Resource: "extensionconfigs",
			},
			ops: createUpdateOps,
		},
		{
			builder: WebhookBuilder{
				Name:     "validation.machinepool.cluster.x-k8s.io",
				Path:     "/validate-cluster-x-k8s-io-v1beta1-machinepool",
				APIGroup: "cluster.x-k8s.io",
				Version:  "v1beta1",
				Resource: "machinepools",
			},
			ops: createUpdateOps,
		},
		{
			builder: WebhookBuilder{
				Name:     "validation.ipaddress.ipam.cluster.x-k8s.io",
				Path:     "/validate-ipam-cluster-x-k8s-io-v1beta1-ipaddress",
				APIGroup: "ipam.cluster.x-k8s.io",
				Version:  "v1beta1",
				Resource: "ipaddresses",
			},
			ops: createUpdateDeleteOps,
		},
		{
			builder: WebhookBuilder{
				Name:     "validation.ipaddressclaim.ipam.cluster.x-k8s.io",
				Path:     "/validate-ipam-cluster-x-k8s-io-v1beta1-ipaddressclaim",
				APIGroup: "ipam.cluster.x-k8s.io",
				Version:  "v1beta1",
				Resource: "ipaddressclaims",
			},
			ops: createUpdateDeleteOps,
		},
	}

	var validatingWebhooks []admissionregistrationv1.ValidatingWebhook
	for _, w := range webhooks {
		webhook := admissionregistrationv1.ValidatingWebhook{
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
		}
		validatingWebhooks = append(validatingWebhooks, webhook)
	}

	return validatingWebhooks
}
