/*
Copyright 2025.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package controllers

import (
	"context"
	"fmt"
	"time"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	rbacv1 "k8s.io/api/rbac/v1"
	apiextensionsv1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/log"

	"github.com/go-openapi/swag"
	capiv1alpha1 "github.com/openshift/oci-capi-operator/api/v1alpha1"
)

// OCIClusterAutoscalerReconciler reconciles a OCIClusterAutoscaler object
type OCIClusterAutoscalerReconciler struct {
	client.Client
	Scheme *runtime.Scheme
}

// +kubebuilder:rbac:groups=capi.openshift.io,resources=ociclusterautoscalers,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=capi.openshift.io,resources=ociclusterautoscalers/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=capi.openshift.io,resources=ociclusterautoscalers/finalizers,verbs=update
// +kubebuilder:rbac:groups=apps,resources=deployments,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=core,resources=serviceaccounts,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=core,resources=secrets,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=core,resources=configmaps,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=rbac.authorization.k8s.io,resources=clusterroles,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=rbac.authorization.k8s.io,resources=clusterrolebindings,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=rbac.authorization.k8s.io,resources=roles,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=rbac.authorization.k8s.io,resources=rolebindings,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=apiextensions.k8s.io,resources=customresourcedefinitions,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=security.openshift.io,resources=securitycontextconstraints,verbs=get;list;watch;create;update;patch;delete

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
func (r *OCIClusterAutoscalerReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	logger := log.FromContext(ctx)

	// Fetch the OCIClusterAutoscaler instance
	autoscaler := &capiv1alpha1.OCIClusterAutoscaler{}
	err := r.Get(ctx, req.NamespacedName, autoscaler)
	if err != nil {
		if errors.IsNotFound(err) {
			logger.Info("OCIClusterAutoscaler resource not found. Ignoring since object must be deleted")
			return ctrl.Result{}, nil
		}
		logger.Error(err, "Failed to get OCIClusterAutoscaler")
		return ctrl.Result{}, err
	}

	// Initialize status if not set
	if autoscaler.Status.Phase == "" {
		autoscaler.Status.Phase = "Initializing"
		if err := r.Status().Update(ctx, autoscaler); err != nil {
			return ctrl.Result{}, err
		}
	}

	// Set up finalizer
	finalizerName := "ociclusterautoscaler.capi.openshift.io/finalizer"
	if autoscaler.ObjectMeta.DeletionTimestamp.IsZero() {
		if !controllerutil.ContainsFinalizer(autoscaler, finalizerName) {
			controllerutil.AddFinalizer(autoscaler, finalizerName)
			return ctrl.Result{}, r.Update(ctx, autoscaler)
		}
	} else {
		if controllerutil.ContainsFinalizer(autoscaler, finalizerName) {
			// Perform cleanup
			if err := r.cleanup(ctx, autoscaler); err != nil {
				return ctrl.Result{}, err
			}
			controllerutil.RemoveFinalizer(autoscaler, finalizerName)
			return ctrl.Result{}, r.Update(ctx, autoscaler)
		}
		return ctrl.Result{}, nil
	}

	// Reconcile the OCI CAPI stack
	result, err := r.reconcileOCICapiStack(ctx, autoscaler)
	if err != nil {
		// Update status with error condition
		meta.SetStatusCondition(&autoscaler.Status.Conditions, metav1.Condition{
			Type:    "Ready",
			Status:  metav1.ConditionFalse,
			Reason:  "ReconcileError",
			Message: err.Error(),
		})
		autoscaler.Status.Phase = "Error"
		r.Status().Update(ctx, autoscaler)
		return result, err
	}

	// Update status
	autoscaler.Status.ObservedGeneration = autoscaler.Generation
	if autoscaler.Status.CAPIInstalled && autoscaler.Status.ClusterAutoscalerDeployed {
		autoscaler.Status.Phase = "Ready"
		meta.SetStatusCondition(&autoscaler.Status.Conditions, metav1.Condition{
			Type:    "Ready",
			Status:  metav1.ConditionTrue,
			Reason:  "ReconcileSuccess",
			Message: "OCI CAPI autoscaler is ready",
		})
	}

	if err := r.Status().Update(ctx, autoscaler); err != nil {
		return ctrl.Result{}, err
	}

	return result, nil
}

func (r *OCIClusterAutoscalerReconciler) reconcileOCICapiStack(ctx context.Context, autoscaler *capiv1alpha1.OCIClusterAutoscaler) (ctrl.Result, error) {
	logger := log.FromContext(ctx)

	// Step 0: Validate the autoscaler spec
	if err := validate(autoscaler); err != nil {
		logger.Error(err, "Invalid autoscaler spec")
		return ctrl.Result{}, err
	}

	// Step 1: Ensure required namespaces exist
	if err := r.ensureNamespaces(ctx, autoscaler); err != nil {
		logger.Error(err, "Failed to ensure namespaces")
		return ctrl.Result{}, err
	}

	// Step 2: Create SecurityContextConstraints for CAPI
	if err := r.createSecurityContextConstraints(ctx, autoscaler); err != nil {
		logger.Error(err, "Failed to create SecurityContextConstraints")
		return ctrl.Result{}, err
	}

	// Step 3: Create OCI credentials secret
	if err := r.createOCICredentialsSecret(ctx, autoscaler); err != nil {
		logger.Error(err, "Failed to create OCI credentials secret")
		return ctrl.Result{}, err
	}

	// Step 4: Check if CAPI CRDs exist (assuming clusterctl is used externally)
	capiInstalled, err := r.checkCAPIInstallation(ctx)
	if err != nil {
		logger.Error(err, "Failed to check CAPI installation")
		return ctrl.Result{RequeueAfter: time.Minute * 5}, err
	}
	autoscaler.Status.CAPIInstalled = capiInstalled

	if !capiInstalled {
		logger.Info("CAPI is not installed, waiting...")
		return ctrl.Result{RequeueAfter: time.Minute * 2}, nil
	}

	// Step 5: Deploy cluster-autoscaler
	if err := r.deployClusterAutoscaler(ctx, autoscaler); err != nil {
		logger.Error(err, "Failed to deploy cluster-autoscaler")
		return ctrl.Result{}, err
	}
	autoscaler.Status.ClusterAutoscalerDeployed = true

	// Step 6: Create additional RBAC for cluster-autoscaler
	if err := r.createClusterAutoscalerRBAC(ctx, autoscaler); err != nil {
		logger.Error(err, "Failed to create cluster-autoscaler RBAC")
		return ctrl.Result{}, err
	}

	return ctrl.Result{RequeueAfter: time.Minute * 10}, nil
}

func (r *OCIClusterAutoscalerReconciler) ensureNamespaces(ctx context.Context, autoscaler *capiv1alpha1.OCIClusterAutoscaler) error {
	namespaces := []string{
		"cluster-api-provider-oci-system",
		capiSystemNamespace,
	}

	for _, name := range namespaces {
		ns := &corev1.Namespace{
			ObjectMeta: metav1.ObjectMeta{
				Name: name,
			},
		}
		_, err := controllerutil.CreateOrUpdate(ctx, r.Client, ns, func() error {
			return nil
		})
		if err != nil {
			return fmt.Errorf("failed to ensure namespace %s: %w", name, err)
		}
	}
	return nil
}

func (r *OCIClusterAutoscalerReconciler) cleanup(ctx context.Context, autoscaler *capiv1alpha1.OCIClusterAutoscaler) error {
	// Cleanup resources created by the operator
	// This would include removing deployments, RBAC, secrets, etc.
	return nil
}

func (r *OCIClusterAutoscalerReconciler) createSecurityContextConstraintsCAPI(ctx context.Context, autoscaler *capiv1alpha1.OCIClusterAutoscaler) error {
	// This would create the SCC needed for CAPI components
	// Implementation depends on OpenShift security API
	return nil
}

func (r *OCIClusterAutoscalerReconciler) createOCICredentialsSecret(ctx context.Context, autoscaler *capiv1alpha1.OCIClusterAutoscaler) error {
	// Get private key from referenced secret first
	privateKeySecret := &corev1.Secret{}
	err := r.Get(ctx, types.NamespacedName{
		Name:      autoscaler.Spec.OCI.PrivateKeySecretRef.Name,
		Namespace: autoscaler.Namespace,
	}, privateKeySecret)
	if err != nil {
		return fmt.Errorf("failed to get private key secret: %w", err)
	}

	keyName := autoscaler.Spec.OCI.PrivateKeySecretRef.Key
	if keyName == "" {
		keyName = "private_key"
	}
	privateKey, exists := privateKeySecret.Data[keyName]
	if !exists {
		return fmt.Errorf("private key not found in secret %s with key %s", autoscaler.Spec.OCI.PrivateKeySecretRef.Name, keyName)
	}

	// Create or update secret with OCI credentials for CAPI
	secretName := "oci-credentials"
	namespace := "cluster-api-provider-oci-system"

	secret := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      secretName,
			Namespace: namespace,
		},
	}

	_, err = controllerutil.CreateOrUpdate(ctx, r.Client, secret, func() error {
		secret.Type = corev1.SecretTypeOpaque
		secret.Data = map[string][]byte{
			"tenancy":     []byte(autoscaler.Spec.OCI.TenancyID),
			"user":        []byte(autoscaler.Spec.OCI.UserID),
			"region":      []byte(autoscaler.Spec.OCI.Region),
			"fingerprint": []byte(autoscaler.Spec.OCI.Fingerprint),
			"key":         privateKey,
		}
		return controllerutil.SetControllerReference(autoscaler, secret, r.Scheme)
	})

	return err
}

func (r *OCIClusterAutoscalerReconciler) checkCAPIInstallation(ctx context.Context) (bool, error) {
	// Check if CAPI CRDs exist
	crd := &apiextensionsv1.CustomResourceDefinition{}
	err := r.Get(ctx, types.NamespacedName{Name: "clusters.cluster.x-k8s.io"}, crd)
	return err == nil, nil
}

func (r *OCIClusterAutoscalerReconciler) deployClusterAutoscaler(ctx context.Context, autoscaler *capiv1alpha1.OCIClusterAutoscaler) error {
	// Create or update service account
	sa := &corev1.ServiceAccount{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "oci-cluster-autoscaler",
			Namespace: capiSystemNamespace,
		},
	}
	_, err := controllerutil.CreateOrUpdate(ctx, r.Client, sa, func() error {
		return controllerutil.SetControllerReference(autoscaler, sa, r.Scheme)
	})
	if err != nil {
		return err
	}

	// Create or update deployment
	image := ClusterAutoscalerImage
	if autoscaler.Spec.ClusterAutoscaler.Image != "" {
		image = autoscaler.Spec.ClusterAutoscaler.Image
	}

	deployment := &appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "oci-cluster-autoscaler",
			Namespace: capiSystemNamespace,
		},
	}

	_, err = controllerutil.CreateOrUpdate(ctx, r.Client, deployment, func() error {
		deployment.Spec = appsv1.DeploymentSpec{
			Replicas: swag.Int32(1),
			Selector: &metav1.LabelSelector{
				MatchLabels: map[string]string{
					"app": "oci-cluster-autoscaler",
				},
			},
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Labels: map[string]string{
						"app": "oci-cluster-autoscaler",
					},
				},
				Spec: corev1.PodSpec{
					ServiceAccountName: "oci-cluster-autoscaler",
					Containers: []corev1.Container{
						{
							Name:  "cluster-autoscaler",
							Image: image,
							Command: []string{
								"./cluster-autoscaler",
								"--v=4",
								"--stderrthreshold=info",
								"--cloud-provider=clusterapi",
								"--namespace=" + capiSystemNamespace,
								"--clusterapi-cloud-config-authoritative",
								"--node-group-auto-discovery=clusterapi:namespace=" + capiSystemNamespace,
							},
						},
					},
				},
			},
		}
		return controllerutil.SetControllerReference(autoscaler, deployment, r.Scheme)
	})

	return err
}

func (r *OCIClusterAutoscalerReconciler) createClusterAutoscalerRBAC(ctx context.Context, autoscaler *capiv1alpha1.OCIClusterAutoscaler) error {
	// Create or update ClusterRole for additional CAPI permissions
	clusterRole := &rbacv1.ClusterRole{
		ObjectMeta: metav1.ObjectMeta{
			Name: "oci-cluster-autoscaler-extra",
		},
	}

	_, err := controllerutil.CreateOrUpdate(ctx, r.Client, clusterRole, func() error {
		clusterRole.Rules = []rbacv1.PolicyRule{
			{
				APIGroups: []string{"infrastructure.cluster.x-k8s.io"},
				Resources: []string{"*"},
				Verbs:     []string{"get", "list", "watch", "update"},
			},
			{
				APIGroups: []string{"cluster.x-k8s.io"},
				Resources: []string{"*"},
				Verbs:     []string{"get", "list", "watch", "update"},
			},
		}
		return nil
	})
	if err != nil {
		return err
	}

	// Create or update ClusterRoleBinding
	clusterRoleBinding := &rbacv1.ClusterRoleBinding{
		ObjectMeta: metav1.ObjectMeta{
			Name: "oci-cluster-autoscaler-extra",
		},
	}

	_, err = controllerutil.CreateOrUpdate(ctx, r.Client, clusterRoleBinding, func() error {
		clusterRoleBinding.RoleRef = rbacv1.RoleRef{
			APIGroup: "rbac.authorization.k8s.io",
			Kind:     "ClusterRole",
			Name:     "oci-cluster-autoscaler-extra",
		}
		clusterRoleBinding.Subjects = []rbacv1.Subject{
			{
				Kind:      "ServiceAccount",
				Name:      "oci-cluster-autoscaler",
				Namespace: capiSystemNamespace,
			},
		}
		return nil
	})

	return err
}

func validate(instance *capiv1alpha1.OCIClusterAutoscaler) error {
	if err := validateAutoscalerSpec(&instance.Spec); err != nil {
		return fmt.Errorf("invalid autoscaler spec: %w", err)
	}
	return nil
}

func validateAutoscalerSpec(spec *capiv1alpha1.OCIClusterAutoscalerSpec) error {
	if spec.Autoscaling.MinNodes > spec.Autoscaling.MaxNodes {
		return fmt.Errorf("minNodes [%d] must be less than or equal to maxNodes [%d]", spec.Autoscaling.MinNodes, spec.Autoscaling.MaxNodes)
	}

	if spec.Autoscaling.Shape == "" {
		return fmt.Errorf("shape is required")
	}

	return nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *OCIClusterAutoscalerReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&capiv1alpha1.OCIClusterAutoscaler{}).
		Complete(r)
}
