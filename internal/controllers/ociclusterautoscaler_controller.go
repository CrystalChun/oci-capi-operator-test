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

	"github.com/go-logr/logr"
	capiv1alpha1 "github.com/openshift/oci-capi-operator/api/v1alpha1"
	"github.com/openshift/oci-capi-operator/internal/components"
	"github.com/openshift/oci-capi-operator/internal/components/autoscaler"
	"github.com/openshift/oci-capi-operator/internal/components/capi"
	"github.com/openshift/oci-capi-operator/internal/components/capoci"
	enableautoscaler "github.com/openshift/oci-capi-operator/internal/components/enable_autoscaler"

	"github.com/openshift/oci-capi-operator/internal/utils"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/client-go/rest"

	securityv1 "github.com/openshift/api/security/v1"

	admissionregistrationv1 "k8s.io/api/admissionregistration/v1"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	rbacv1 "k8s.io/api/rbac/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/log"
)

// OCIClusterAutoscalerReconciler reconciles a OCIClusterAutoscaler object
type OCIClusterAutoscalerReconciler struct {
	RestConfig *rest.Config
	client.Client
	Scheme            *runtime.Scheme
	CAPOCICredentials capoci.CAPOCICredentials
	AutoScalingConfig enableautoscaler.Config
}

// +kubebuilder:rbac:groups=capi.openshift.io,resources=ociclusterautoscalers,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=capi.openshift.io,resources=ociclusterautoscalers/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=capi.openshift.io,resources=ociclusterautoscalers/finalizers,verbs=update
// +kubebuilder:rbac:groups=cluster.x-k8s.io,resources=clusters,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=cluster.x-k8s.io,resources=clusters/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=cluster.x-k8s.io,resources=clusters/finalizers,verbs=update
// +kubebuilder:rbac:groups=cluster.x-k8s.io,resources=machinetemplates,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=cluster.x-k8s.io,resources=machinetemplates/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=cluster.x-k8s.io,resources=machinedeployments,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=cluster.x-k8s.io,resources=machinedeployments/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=cluster.x-k8s.io,resources=machinedeployments/finalizers,verbs=update
// +kubebuilder:rbac:groups=infrastructure.cluster.x-k8s.io,resources=ocicluster/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=apps,resources=deployments,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=core,resources=namespaces,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=core,resources=serviceaccounts,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=core,resources=secrets,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=core,resources=configmaps,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=rbac.authorization.k8s.io,resources=clusterroles,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=rbac.authorization.k8s.io,resources=clusterroles/aggregation,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=rbac.authorization.k8s.io,resources=clusterrolebindings,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=rbac.authorization.k8s.io,resources=roles,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=rbac.authorization.k8s.io,resources=rolebindings,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=apiextensions.k8s.io,resources=customresourcedefinitions,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=apiextensions.k8s.io,resources=customresourcedefinitions/status,verbs=get;patch;update
// +kubebuilder:rbac:groups=security.openshift.io,resources=securitycontextconstraints,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=admissionregistration.k8s.io,resources=validatingwebhookconfigurations,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=admissionregistration.k8s.io,resources=mutatingwebhookconfigurations,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=core,resources=services,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=coordination.k8s.io,resources=leases,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=core,resources=events,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=addons.cluster.x-k8s.io,resources=clusterresourcesets/finalizers;clusterresourcesets/status,verbs=get;patch;update
// +kubebuilder:rbac:groups=addons.cluster.x-k8s.io;bootstrap.cluster.x-k8s.io;controlplane.cluster.x-k8s.io;infrastructure.cluster.x-k8s.io,resources=*,verbs=create;delete;get;list;patch;update;watch
// +kubebuilder:rbac:groups=authentication.k8s.io,resources=tokenreviews,verbs=create
// +kubebuilder:rbac:groups=authorization.k8s.io,resources=subjectaccessreviews,verbs=create
// +kubebuilder:rbac:groups=cluster.x-k8s.io,resources=clusterclasses;clusterclasses/status;clusters;clusters/finalizers;clusters/status;machinedrainrules;machinehealthchecks/finalizers;machinehealthchecks/status,verbs=get;list;patch;update;watch
// +kubebuilder:rbac:groups=cluster.x-k8s.io,resources=machinedeployments;machinedeployments/finalizers;machinedeployments/status;machinehealthchecks;machinepools;machinepools/finalizers;machinepools/status;machines;machines/finalizers;machines/status;machinesets;machinesets/finalizers;machinesets/status,verbs=create;delete;get;list;patch;update;watch
// +kubebuilder:rbac:groups=ipam.cluster.x-k8s.io,resources=ipaddressclaims;ipaddresses,verbs=get;list;patch;update;watch
// +kubebuilder:rbac:groups=ipam.cluster.x-k8s.io,resources=ipaddressclaims/status,verbs=patch;update
// +kubebuilder:rbac:groups=runtime.cluster.x-k8s.io,resources=extensionconfigs;extensionconfigs/status,verbs=get;list;patch;update;watch
// +kubebuilder:rbac:groups=infrastructure.cluster.x-k8s.io,resources=ocimachines,verbs=get;list;watch
// +kubebuilder:rbac:groups=config.openshift.io,resources=infrastructures,verbs=get;list;watch

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
				logger.Error(err, "Failed to cleanup")
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
	if err := validate(instance, r.AutoScalingConfig); err != nil {
		logger.Error(err, "Invalid autoscaler spec")
		return ctrl.Result{}, err
	}

	// Step 1: Reconcile CAPI components
	capiComponent := capi.GetComponents(CAPISystemNamespace, CAPOCISystemNamespace, CAPIServiceAccountName, CAPOCIServiceAccountName, CAPIClusterRoleBindingName, instance)
	err := reconcileComponents(ctx, r.Client, capiComponent)
	if err != nil {
		logger.Error(err, "Failed to reconcile CAPI components")
		return ctrl.Result{}, err
	}
	logger.Info("CAPI components created")

	capiComponents, err := capi.GetClusterctlComponents(ctx, CAPIDeploymentName, CAPIServiceAccountName, CAPISystemNamespace, instance, CAPIWebhookServiceName, r.Scheme)
	if err != nil {
		logger.Error(err, "Failed to create CAPI components")
		return ctrl.Result{}, err
	}

	err = reconcileClusterctlComponents(ctx, r.Client, capiComponents)
	if err != nil {
		logger.Error(err, "Failed to reconcile CAPI clusterctl components")
		return ctrl.Result{}, err
	}
	logger.Info("CAPI Clusterctl components created")

	// Step 2: Reconcile CAPOCI components
	capociComponent := capoci.GetComponents(CAPOCISystemNamespace, instance, &r.CAPOCICredentials)
	err = reconcileComponents(ctx, r.Client, capociComponent)
	if err != nil {
		logger.Error(err, "Failed to reconcile CAPOCI components")
		return ctrl.Result{RequeueAfter: time.Second * 20}, nil
	}
	logger.Info("CAPOCI components created")

	capociComponents, err := capoci.GetClusterctlComponents(ctx, CAPOCIDeploymentName, CAPOCIServiceAccountName, CAPOCISystemNamespace, instance, CAPOCIWebhookServiceName, r.Scheme)
	if err != nil {
		logger.Error(err, "Failed to create CAPOCI components")
		return ctrl.Result{}, err
	}
	err = reconcileClusterctlComponents(ctx, r.Client, capociComponents)
	if err != nil {
		logger.Error(err, "Failed to reconcile CAPOCI clusterctl components")
		return ctrl.Result{}, err
	}
	logger.Info("CAPOCI clusterctl components created")

	// Step 3: Check if CAPI is already deployed
	capiInstalled, err := r.checkCAPIInstallation(ctx, instance)
	if err != nil && !errors.IsNotFound(err) {
		logger.Info("Failed to check CAPI installation", "error", err)
		return ctrl.Result{RequeueAfter: time.Second * 20}, nil
	}
	instance.Status.CAPIInstalled = capiInstalled

	if !capiInstalled {
		logger.Info("CAPI is not installed, requeuing...")
		return ctrl.Result{RequeueAfter: time.Second * 20}, nil
	}
	logger.Info("CAPI is installed")

	// Step 4: Reconcile Cluster Autoscaler components
	autoscalerDeploymentValues := getAutoscalerDeploymentValues(instance)

	autoscalerComponents := autoscaler.GetComponents(&autoscalerDeploymentValues, instance, r.Scheme)

	err = reconcileComponents(ctx, r.Client, autoscalerComponents)
	if err != nil {
		logger.Error(err, "Failed to reconcile autoscaler components")
		return ctrl.Result{}, err
	}
	logger.Info("Autoscaler components created", "components", autoscalerComponents)

	// Install the autoscaler Helm chart
	err = autoscaler.InstallAutoscaler(instance, &autoscalerDeploymentValues, r.RestConfig)
	if err != nil {
		logger.Error(err, "Failed to install autoscaler Helm chart")
		return ctrl.Result{}, err
	}
	logger.Info("Helm chart installed")
	instance.Status.ClusterAutoscalerDeployed = true

	// Step 5: Reconcile Enable Autoscaler components
	clusterName, err := utils.GetClusterName(ctx, r.Client)
	if err != nil {
		logger.Error(err, "Failed to get cluster name")
		return ctrl.Result{}, err
	}

	secret, err := utils.GetSecret(ctx, r.Client, fmt.Sprintf("%s-token", CAPIServiceAccountName), CAPISystemNamespace)
	if err != nil {
		logger.Error(err, "Failed to get CAPI service account secret")
		return ctrl.Result{}, err
	}
	logger.Info("CAPI service account secret", "secret", secret)

	autoscalerConfig, err := enableautoscaler.SetAutoScalingConfig(ctx, r.Client, instance, r.AutoScalingConfig)
	if err != nil {
		logger.Error(err, "Failed to set autoscaler config")
		return ctrl.Result{}, err
	}

	enableAutoscalerComponent := enableautoscaler.GetComponents(ctx, r.Client, CAPISystemNamespace, clusterName, CAPIServiceAccountName, instance, autoscalerConfig)
	err = reconcileComponents(ctx, r.Client, enableAutoscalerComponent)
	if err != nil {
		logger.Error(err, "Failed to reconcile Enable Autoscaler components")
		return ctrl.Result{}, err
	}
	logger.Info("Enable Autoscaler components created")
	return ctrl.Result{}, nil
}

func (r *OCIClusterAutoscalerReconciler) cleanup(ctx context.Context, instance *capiv1alpha1.OCIClusterAutoscaler) error {
	logger := log.FromContext(ctx)
	logger.Info("Starting cleanup of all resources")
	autoscalerValues := getAutoscalerDeploymentValues(instance)
	clusterName, err := utils.GetClusterName(ctx, r.Client)
	if err != nil {
		logger.Error(err, "Failed to get cluster name")
		return err
	}
	autoscalerComponents := enableautoscaler.GetComponents(ctx, r.Client, CAPISystemNamespace, clusterName, CAPIServiceAccountName, instance, r.AutoScalingConfig)

	err = removeComponent(ctx, r.Client, autoscalerComponents)
	if err != nil {
		logger.Error(err, "Failed to remove enable autoscaler components")
		return err
	}
	logger.Info("Enable autoscaler components removed")

	err = autoscaler.RemoveAutoscaler(&autoscalerValues, r.RestConfig)
	if err != nil {
		logger.Error(err, "Failed to remove autoscaler Helm chart")
		return err
	}
	logger.Info("Autoscaler Helm chart removed")

	capociComponents, err := capoci.GetClusterctlComponents(ctx, CAPOCIDeploymentName, CAPOCIServiceAccountName, CAPOCISystemNamespace, instance, CAPOCIWebhookServiceName, r.Scheme)
	if err != nil {
		logger.Error(err, "Failed to get CAPOCI clusterctl components")
		return err
	}
	err = removeClusterctlComponents(ctx, r.Client, capociComponents)
	if err != nil {
		logger.Error(err, "Failed to remove CAPOCI clusterctl components")
		return err
	}
	logger.Info("CAPOCI clusterctl components removed")

	capociComponent := capoci.GetComponents(CAPOCISystemNamespace, instance, &r.CAPOCICredentials)
	err = removeComponent(ctx, r.Client, capociComponent)
	if err != nil {
		logger.Error(err, "Failed to remove CAPOCI components")
		return err
	}
	logger.Info("CAPOCI components removed")

	capiComponents, err := capi.GetClusterctlComponents(ctx, CAPIDeploymentName, CAPIServiceAccountName, CAPISystemNamespace, instance, CAPIWebhookServiceName, r.Scheme)
	if err != nil {
		logger.Error(err, "Failed to get CAPI clusterctl components")
		return err
	}
	err = removeClusterctlComponents(ctx, r.Client, capiComponents)
	if err != nil {
		logger.Error(err, "Failed to remove CAPI clusterctl components")
		return err
	}
	logger.Info("CAPI clusterctl components removed")

	capiComponent := capi.GetComponents(CAPISystemNamespace, CAPOCISystemNamespace, CAPIServiceAccountName, CAPOCIServiceAccountName, CAPIClusterRoleBindingName, instance)
	err = removeComponent(ctx, r.Client, capiComponent)
	if err != nil {
		logger.Error(err, "Failed to remove CAPI components")
		return err
	}
	logger.Info("CAPI components removed")
	/* 	// Define the resource types we need to clean up based on RBAC rules and SetupWithManager
	   	gvks := []struct {
	   		list     client.ObjectList
	   		resource string
	   	}{
	   		{&corev1.NamespaceList{}, "namespaces"},
	   		{&corev1.ServiceAccountList{}, "serviceaccounts"},
	   		{&corev1.SecretList{}, "secrets"},
	   		{&corev1.ConfigMapList{}, "configmaps"},
	   		{&corev1.ServiceList{}, "services"},
	   		{&appsv1.DeploymentList{}, "deployments"},
	   		{&rbacv1.ClusterRoleList{}, "clusterroles"},
	   		{&rbacv1.ClusterRoleBindingList{}, "clusterrolebindings"},
	   		{&rbacv1.RoleList{}, "roles"},
	   		{&rbacv1.RoleBindingList{}, "rolebindings"},
	   		{&admissionregistrationv1.ValidatingWebhookConfigurationList{}, "validatingwebhookconfigurations"},
	   		{&admissionregistrationv1.MutatingWebhookConfigurationList{}, "mutatingwebhookconfigurations"},
	   		{&securityv1.SecurityContextConstraintsList{}, "securitycontextconstraints"},
	   	}

	   	// Label selector for resources managed by this controller
	   	labelSelector := labels.SelectorFromSet(map[string]string{ManagedByLabel: instance.Name})

	   	// Namespaces to check
	   	namespaces := []string{CAPISystemNamespace, CAPOCISystemNamespace}

	   	// Delete resources in each namespace
	   	for _, ns := range namespaces {
	   		logger.Info("Cleaning up resources in namespace", "namespace", ns)

	   		for _, gvk := range gvks {
	   			logger.Info("Listing resources", "resource", gvk.resource, "namespace", ns)

	   			// Skip namespace-scoped list for cluster-scoped resources
	   			if gvk.resource == "clusterroles" ||
	   				gvk.resource == "clusterrolebindings" ||
	   				gvk.resource == "validatingwebhookconfigurations" ||
	   				gvk.resource == "mutatingwebhookconfigurations" ||
	   				gvk.resource == "securitycontextconstraints" {
	   				continue
	   			}

	   			// List resources
	   			err := r.List(ctx, gvk.list, &client.ListOptions{
	   				Namespace:     ns,
	   				LabelSelector: labelSelector,
	   			})
	   			if err != nil {
	   				logger.Error(err, "Failed to list resources", "resource", gvk.resource, "namespace", ns)
	   				return err
	   			}

	   			// Delete each resource
	   			if err := deleteResourceList(ctx, r.Client, gvk.list, logger); err != nil {
	   				return err
	   			}
	   		}
	   	}

	   	// Delete cluster-scoped resources
	   	logger.Info("Cleaning up cluster-scoped resources")
	   	for _, gvk := range gvks {
	   		// Only process cluster-scoped resources
	   		if gvk.resource != "clusterroles" &&
	   			gvk.resource != "clusterrolebindings" &&
	   			gvk.resource != "validatingwebhookconfigurations" &&
	   			gvk.resource != "mutatingwebhookconfigurations" &&
	   			gvk.resource != "securitycontextconstraints" {
	   			continue
	   		}

	   		logger.Info("Listing cluster-scoped resources", "resource", gvk.resource)

	   		// List resources
	   		err := r.List(ctx, gvk.list, &client.ListOptions{
	   			LabelSelector: labelSelector,
	   		})
	   		if err != nil {
	   			logger.Error(err, "Failed to list cluster-scoped resources", "resource", gvk.resource)
	   			return err
	   		}

	   		// Delete each resource
	   		if err := deleteResourceList(ctx, r.Client, gvk.list, logger); err != nil {
	   			return err
	   		}
	   	} */

	// Remove the Autoscaler Helm chart

	logger.Info("Cleanup completed successfully")
	return nil
}

// deleteResourceList is a helper function to delete all resources in a list
func deleteResourceList(ctx context.Context, c client.Client, list client.ObjectList, logger logr.Logger) error {
	items, err := meta.ExtractList(list)
	if err != nil {
		return err
	}

	for _, item := range items {
		obj, ok := item.(client.Object)
		if !ok {
			continue
		}

		logger.Info("Deleting resource", "name", obj.GetName(), "namespace", obj.GetNamespace(), "kind", obj.GetObjectKind().GroupVersionKind().Kind)
		if err := c.Delete(ctx, obj); err != nil && !errors.IsNotFound(err) {
			logger.Error(err, "Failed to delete resource", "name", obj.GetName(), "namespace", obj.GetNamespace())
			return err
		}
	}
	return nil
}

func (r *OCIClusterAutoscalerReconciler) checkCAPIInstallation(ctx context.Context, instance *capiv1alpha1.OCIClusterAutoscaler) (bool, error) {
	// Check if CAPI controller manager is deployed
	capiDeployments := &appsv1.DeploymentList{}
	err := r.List(ctx, capiDeployments, &client.ListOptions{
		Namespace: CAPISystemNamespace,
		LabelSelector: labels.SelectorFromSet(map[string]string{
			ManagedByLabel: instance.Name,
		}),
	})
	if err != nil {
		return false, err
	}

	if len(capiDeployments.Items) == 0 {
		return false, fmt.Errorf("CAPI controller manager is not deployed")
	}

	condition := utils.GetDeploymentCondition(capiDeployments.Items[0].Status.Conditions, appsv1.DeploymentAvailable)
	if condition == nil {
		return false, fmt.Errorf("CAPI controller manager is not available")
	}

	if condition.Status != corev1.ConditionTrue {
		return false, fmt.Errorf("CAPI controller manager is not available")
	}

	return true, nil
}

func validate(instance *capiv1alpha1.OCIClusterAutoscaler, config enableautoscaler.Config) error {
	if err := enableautoscaler.ValidateMinMaxNodes(instance, config); err != nil {
		return fmt.Errorf("invalid Min/Max nodes set in either the autoscaler spec or the config: %w", err)
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

func reconcileComponents(ctx context.Context, client client.Client, components *components.Component) error {
	allErrs := []error{}
	logger := log.FromContext(ctx)
	//logger.Info("Reconciling components", "component", components.Name, "subcomponents", components.Subcomponents)
	for _, component := range components.Subcomponents {
		//logger.Info("Reconciling subcomponent", "subcomponent", component.Name)

		_, err := controllerutil.CreateOrPatch(ctx, client, component.Object, component.MutateFn)
		if err != nil {
			logger.Error(err, "Failed to reconcile component", "component", component.Name)
			allErrs = append(allErrs, err)
			continue
		}
		//fmt.Printf("Component %s operation: %s\n", component.Name, op)
	}
	if len(allErrs) > 0 {
		return fmt.Errorf("failed to reconcile components: %v", allErrs)
	}
	return nil
}
func reconcileClusterctlComponents(ctx context.Context, client client.Client, components []unstructured.Unstructured) error {
	for i := range components {
		component := components[i].DeepCopy()
		_, err := controllerutil.CreateOrPatch(ctx, client, component, func() error {
			// Get the current object to update
			current := component.DeepCopy()

			// Copy the spec and other relevant fields from our desired state
			if spec, exists, err := unstructured.NestedFieldCopy(components[i].Object, "spec"); err == nil && exists {
				if err := unstructured.SetNestedField(current.Object, spec, "spec"); err != nil {
					return err
				}
			}

			// Copy metadata fields we want to preserve/update
			if annotations := components[i].GetAnnotations(); len(annotations) > 0 {
				current.SetAnnotations(annotations)
			}
			if labels := components[i].GetLabels(); len(labels) > 0 {
				current.SetLabels(labels)
			}

			// Copy the object's content
			component.Object = current.Object
			return nil
		})
		if err != nil && !errors.IsAlreadyExists(err) {
			return err
		}
		//fmt.Printf("Component %s operation: %s\n", component.GetName(), op)
	}
	return nil
}

func removeComponent(ctx context.Context, client client.Client, component *components.Component) error {
	for _, subcomponent := range component.Subcomponents {
		err := client.Delete(ctx, subcomponent.Object)
		if err != nil && !errors.IsNotFound(err) {
			return err
		}
	}
	return nil
}
func removeClusterctlComponents(ctx context.Context, client client.Client, components []unstructured.Unstructured) error {

	for _, component := range components {
		err := client.Delete(ctx, &component)
		if err != nil {
			return err
		}
	}
	return nil
}

func getAutoscalerDeploymentValues(instance *capiv1alpha1.OCIClusterAutoscaler) autoscaler.AutoscalerDeploymentValues {
	return autoscaler.GetAutoscalerDeploymentValues(autoscaler.AutoscalerDeploymentValues{
		Name:                 AutoscalerDeploymentName,
		Namespace:            CAPISystemNamespace,
		CloudProvider:        AutoScalerCloudProvider,
		ServiceAccountName:   AutoscalerDeploymentName,
		RepositoryURL:        AutoscalerRepoURL,
		Chart:                AutoscalerChartName,
		Version:              "9.40.0",
		CreateRBAC:           true,
		CreateServiceAccount: true,
	}, instance)
}
