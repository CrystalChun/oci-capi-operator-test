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

	securityv1 "github.com/openshift/api/security/v1"
	"github.com/openshift/oci-capi-operator/internal/components"
	"github.com/openshift/oci-capi-operator/internal/utils"
	admissionregistrationv1 "k8s.io/api/admissionregistration/v1"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	rbacv1 "k8s.io/api/rbac/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/log"

	capiv1alpha1 "github.com/openshift/oci-capi-operator/api/v1alpha1"
	"github.com/openshift/oci-capi-operator/internal/components/autoscaler"
	"github.com/openshift/oci-capi-operator/internal/components/capi"
	"github.com/openshift/oci-capi-operator/internal/components/capoci"
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
// +kubebuilder:rbac:groups=admissionregistration.k8s.io,resources=validatingwebhookconfigurations,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=admissionregistration.k8s.io,resources=mutatingwebhookconfigurations,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=core,resources=services,verbs=get;list;watch;create;update;patch;delete

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
func (r *OCIClusterAutoscalerReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	logger := log.FromContext(ctx)

	// Fetch the OCIClusterAutoscaler instance
	instance := &capiv1alpha1.OCIClusterAutoscaler{}
	err := r.Get(ctx, req.NamespacedName, instance)
	if err != nil {
		if errors.IsNotFound(err) {
			logger.Info("OCIClusterAutoscaler resource not found. Ignoring since object must be deleted")
			return ctrl.Result{}, nil
		}
		logger.Error(err, "Failed to get OCIClusterAutoscaler")
		return ctrl.Result{}, err
	}

	// Initialize status if not set
	if instance.Status.Phase == "" {
		instance.Status.Phase = "Initializing"
		if err := r.Status().Update(ctx, instance); err != nil {
			return ctrl.Result{}, err
		}
	}

	// Set up finalizer
	if instance.ObjectMeta.DeletionTimestamp.IsZero() {
		if !controllerutil.ContainsFinalizer(instance, FinalizerName) {
			controllerutil.AddFinalizer(instance, FinalizerName)
			return ctrl.Result{}, r.Update(ctx, instance)
		}
	} else {
		if controllerutil.ContainsFinalizer(instance, FinalizerName) {
			// Perform cleanup
			if err := r.cleanup(ctx, instance); err != nil {
				return ctrl.Result{}, err
			}
			controllerutil.RemoveFinalizer(instance, FinalizerName)
			return ctrl.Result{}, r.Update(ctx, instance)
		}
		return ctrl.Result{}, nil
	}

	// Reconcile the OCI CAPI stack
	result, err := r.reconcileOCICapiStack(ctx, instance)
	if err != nil {
		// Update status with error condition
		meta.SetStatusCondition(&instance.Status.Conditions, metav1.Condition{
			Type:    "Ready",
			Status:  metav1.ConditionFalse,
			Reason:  "ReconcileError",
			Message: err.Error(),
		})
		instance.Status.Phase = "Error"
		r.Status().Update(ctx, instance)
		return result, err
	}

	// Update status
	instance.Status.ObservedGeneration = instance.Generation
	if instance.Status.CAPIInstalled && instance.Status.ClusterAutoscalerDeployed {
		instance.Status.Phase = "Ready"
		meta.SetStatusCondition(&instance.Status.Conditions, metav1.Condition{
			Type:    "Ready",
			Status:  metav1.ConditionTrue,
			Reason:  "ReconcileSuccess",
			Message: "OCI CAPI autoscaler is ready",
		})
	}

	if err := r.Status().Update(ctx, instance); err != nil {
		return ctrl.Result{}, err
	}

	return result, nil
}

func (r *OCIClusterAutoscalerReconciler) reconcileOCICapiStack(ctx context.Context, instance *capiv1alpha1.OCIClusterAutoscaler) (ctrl.Result, error) {
	logger := log.FromContext(ctx)

	// Validate the autoscaler spec
	if err := validate(instance); err != nil {
		logger.Error(err, "Invalid autoscaler spec")
		return ctrl.Result{}, err
	}

	// Step 1: Reconcile CAPI components
	capiComponent := capi.NewComponent(CAPISystemNamespace, instance, r.Scheme)
	err := r.reconcileComponent(ctx, capiComponent)
	if err != nil {
		logger.Error(err, "Failed to reconcile CAPI components")
		return ctrl.Result{RequeueAfter: time.Minute * 2}, err
	}

	// Step 2: Reconcile CAPOCI components
	capociComponent := capoci.NewComponent(CAPOCISystemNamespace, instance, r.Scheme)
	err = r.reconcileComponent(ctx, capociComponent)
	if err != nil {
		logger.Error(err, "Failed to reconcile CAPOCI components")
		return ctrl.Result{RequeueAfter: time.Minute * 2}, err
	}

	// Step 3: Check if CAPI is already deployed
	capiInstalled, err := r.checkCAPIInstallation(ctx)
	if err != nil {
		logger.Error(err, "Failed to check CAPI installation")
		return ctrl.Result{RequeueAfter: time.Minute * 5}, err
	}
	instance.Status.CAPIInstalled = capiInstalled

	if !capiInstalled {
		logger.Info("CAPI is not installed, waiting...")
		return ctrl.Result{RequeueAfter: time.Minute * 2}, nil
	}

	// Step 4: Reconcile Cluster Autoscaler components
	autoscalerComponent := autoscaler.NewComponent(CAPISystemNamespace, ClusterAutoscalerImage, instance, r.Scheme)
	err = r.reconcileComponent(ctx, autoscalerComponent)
	if err != nil {
		logger.Error(err, "Failed to reconcile Cluster Autoscaler components")
		return ctrl.Result{RequeueAfter: time.Minute * 2}, err
	}

	instance.Status.ClusterAutoscalerDeployed = true
	return ctrl.Result{RequeueAfter: time.Minute * 10}, nil
}

func (r *OCIClusterAutoscalerReconciler) cleanup(ctx context.Context, instance *capiv1alpha1.OCIClusterAutoscaler) error {
	// Cleanup resources created by the operator
	// This would include removing deployments, RBAC, secrets, etc.
	return nil
}

func (r *OCIClusterAutoscalerReconciler) checkCAPIInstallation(ctx context.Context) (bool, error) {
	// Check if CAPI controller manager is deployed
	capiDeployment := &appsv1.Deployment{}
	err := r.Get(ctx, types.NamespacedName{Name: "cluster-api-controller", Namespace: CAPISystemNamespace}, capiDeployment)
	if err != nil {
		return false, err
	}

	condition := utils.GetDeploymentCondition(capiDeployment.Status.Conditions, appsv1.DeploymentAvailable)
	if condition == nil {
		return false, fmt.Errorf("CAPI controller manager is not available")
	}

	if condition.Status != corev1.ConditionTrue {
		return false, fmt.Errorf("CAPI controller manager is not available")
	}

	return true, nil
}

func (r *OCIClusterAutoscalerReconciler) reconcileComponent(ctx context.Context, component *components.Component) error {
	logger := log.FromContext(ctx)

	logger.Info("Reconciling components of", "component", component.GetName())

	for _, subcomponent := range component.GetSubcomponents() {
		logger.Info("Reconciling subcomponent", "subcomponent", subcomponent.Name)
		obj := subcomponent.Object
		mutateFn := subcomponent.MutateFn
		controllerutil.CreateOrUpdate(ctx, r.Client, obj, mutateFn)
	}
	return nil
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
		Owns(&corev1.Namespace{}).
		Owns(&corev1.ServiceAccount{}).
		Owns(&appsv1.Deployment{}).
		Owns(&securityv1.SecurityContextConstraints{}).
		Owns(&corev1.Secret{}).
		Owns(&rbacv1.ClusterRole{}).
		Owns(&rbacv1.ClusterRoleBinding{}).
		Owns(&admissionregistrationv1.ValidatingWebhookConfiguration{}).
		Owns(&admissionregistrationv1.MutatingWebhookConfiguration{}).
		Owns(&corev1.Service{}).
		Complete(r)
}
