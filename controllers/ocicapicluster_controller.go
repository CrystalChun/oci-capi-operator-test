package controllers

import (
	"context"
	"fmt"

	"github.com/go-logr/logr"
	securityv1 "github.com/openshift/api/security/v1"
	infrastructurev1beta2 "github.com/oracle/cluster-api-provider-oci/api/v1beta2"
	appsv1 "k8s.io/api/apps/v1"
	rbacv1 "k8s.io/api/rbac/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	capiv1beta1 "sigs.k8s.io/cluster-api/api/v1beta1"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"

	ocicapiv1alpha1 "github.com/openshift/oci-capi-operator/api/v1alpha1"
)

// OCICAPIClusterReconciler reconciles a OCICAPICluster object
type OCICAPIClusterReconciler struct {
	client.Client
	Log    logr.Logger
	Scheme *runtime.Scheme
}

//+kubebuilder:rbac:groups=ocicapi.openshift.io,resources=ocicapiclusters,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=ocicapi.openshift.io,resources=ocicapiclusters/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=ocicapi.openshift.io,resources=ocicapiclusters/finalizers,verbs=update
//+kubebuilder:rbac:groups=cluster.x-k8s.io,resources=clusters;machinedeployments,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=infrastructure.cluster.x-k8s.io,resources=ociclusters;ocimachines;ocimachinetemplates,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=security.openshift.io,resources=securitycontextconstraints,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=rbac.authorization.k8s.io,resources=clusterroles;clusterrolebindings,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups="",resources=secrets;serviceaccounts;configmaps,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=certificates.k8s.io,resources=certificatesigningrequests,verbs=get;list;watch;create;update;patch;delete;approve

// Reconcile handles the reconciliation of OCICAPICluster resources
func (r *OCICAPIClusterReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	_ = r.Log.WithValues("ocicapicluster", req.NamespacedName)

	// Fetch the OCICAPICluster instance
	instance := &ocicapiv1alpha1.OCICAPICluster{}
	err := r.Client.Get(ctx, req.NamespacedName, instance)
	if err != nil {
		if apierrors.IsNotFound(err) {
			return ctrl.Result{}, nil
		}
		return ctrl.Result{}, err
	}

	// Initialize the status if needed
	if instance.Status.Phase == "" {
		instance.Status.Phase = "Initializing"
		if err := r.Client.Status().Update(ctx, instance); err != nil {
			return ctrl.Result{}, err
		}
	}

	// Add finalizer
	if !controllerutil.ContainsFinalizer(instance, "ocicapicluster.openshift.io/finalizer") {
		controllerutil.AddFinalizer(instance, "ocicapicluster.openshift.io/finalizer")
		if err := r.Client.Update(ctx, instance); err != nil {
			return ctrl.Result{}, err
		}
	}

	// Handle deletion
	if !instance.DeletionTimestamp.IsZero() {
		return r.reconcileDelete(ctx, instance)
	}

	// Handle reconciliation
	return r.reconcileNormal(ctx, instance)
}

// reconcileNormal handles the main reconciliation logic
func (r *OCICAPIClusterReconciler) reconcileNormal(ctx context.Context, instance *ocicapiv1alpha1.OCICAPICluster) (ctrl.Result, error) {
	// Update status phase
	if instance.Status.Phase == "" || instance.Status.Phase == "Initializing" {
		instance.Status.Phase = "Installing"
		if err := r.Client.Status().Update(ctx, instance); err != nil {
			return ctrl.Result{}, err
		}
	}

	// Step 1: Create required RBAC
	r.Log.Info("Creating RBAC resources")
	if err := r.createOrUpdateSCC(ctx, instance); err != nil {
		r.Log.Error(err, "Failed to create SecurityContextConstraints")
		return ctrl.Result{}, err
	}

	if err := r.createOrUpdateClusterAutoscalerRBAC(ctx, instance); err != nil {
		r.Log.Error(err, "Failed to create cluster-autoscaler RBAC")
		return ctrl.Result{}, err
	}

	// Step 2: Create bootstrap ignition configuration
	r.Log.Info("Creating bootstrap ignition configuration")
	if err := r.createOrUpdateBootstrapIgnition(ctx, instance); err != nil {
		r.Log.Error(err, "Failed to create bootstrap ignition")
		return ctrl.Result{}, err
	}

	// Step 3: Create CAPI cluster configuration
	r.Log.Info("Creating CAPI cluster resources")
	if err := r.createOrUpdateCAPIResources(ctx, instance); err != nil {
		r.Log.Error(err, "Failed to create CAPI resources")
		return ctrl.Result{}, err
	}

	// Step 4: Deploy cluster-autoscaler
	r.Log.Info("Deploying cluster-autoscaler")
	if err := r.createOrUpdateClusterAutoscaler(ctx, instance); err != nil {
		r.Log.Error(err, "Failed to deploy cluster-autoscaler")
		return ctrl.Result{}, err
	}

	// Step 5: Deploy certificate auto-approval component
	r.Log.Info("Deploying certificate approver")
	if err := r.createOrUpdateCertificateApprover(ctx, instance); err != nil {
		r.Log.Error(err, "Failed to deploy certificate approver")
		return ctrl.Result{}, err
	}

	// Update status
	instance.Status.Phase = "Ready"
	instance.Status.ObservedGeneration = instance.Generation
	if err := r.Client.Status().Update(ctx, instance); err != nil {
		return ctrl.Result{}, err
	}

	return ctrl.Result{}, nil
}

// reconcileDelete handles the deletion of the OCICAPICluster
func (r *OCICAPIClusterReconciler) reconcileDelete(ctx context.Context, instance *ocicapiv1alpha1.OCICAPICluster) (ctrl.Result, error) {
	log := r.Log.WithValues("ocicapicluster", fmt.Sprintf("%s/%s", instance.Namespace, instance.Name))

	// Update status to deleting
	if instance.Status.Phase != "Deleting" {
		instance.Status.Phase = "Deleting"
		if err := r.Client.Status().Update(ctx, instance); err != nil {
			return ctrl.Result{}, err
		}
	}

	// Delete cluster-autoscaler deployment
	deploy := &appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "oci-cluster-autoscaler",
			Namespace: capiSystemNamespace,
		},
	}
	if err := r.Client.Delete(ctx, deploy); err != nil && !apierrors.IsNotFound(err) {
		log.Error(err, "Failed to delete cluster-autoscaler deployment")
		return ctrl.Result{}, err
	}

	// Delete certificate approver deployment
	certApprover := &appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "oci-certificate-approver",
			Namespace: capiSystemNamespace,
		},
	}
	if err := r.Delete(ctx, certApprover); err != nil && !apierrors.IsNotFound(err) {
		log.Error(err, "Failed to delete certificate approver deployment")
		return ctrl.Result{}, err
	}

	// Delete CAPI resources
	// MachineDeployment
	machineDeployment := &capiv1beta1.MachineDeployment{
		ObjectMeta: metav1.ObjectMeta{
			Name:      instance.Name,
			Namespace: capiSystemNamespace,
		},
	}
	if err := r.Delete(ctx, machineDeployment); err != nil && !apierrors.IsNotFound(err) {
		log.Error(err, "Failed to delete MachineDeployment")
		return ctrl.Result{}, err
	}

	// OCIMachineTemplate
	machineTemplate := &infrastructurev1beta2.OCIMachineTemplate{
		ObjectMeta: metav1.ObjectMeta{
			Name:      fmt.Sprintf("%s-autoscaling", instance.Name),
			Namespace: capiSystemNamespace,
		},
	}
	if err := r.Delete(ctx, machineTemplate); err != nil && !apierrors.IsNotFound(err) {
		log.Error(err, "Failed to delete OCIMachineTemplate")
		return ctrl.Result{}, err
	}

	// Cluster
	cluster := &capiv1beta1.Cluster{
		ObjectMeta: metav1.ObjectMeta{
			Name:      instance.Name,
			Namespace: capiSystemNamespace,
		},
	}
	if err := r.Delete(ctx, cluster); err != nil && !apierrors.IsNotFound(err) {
		log.Error(err, "Failed to delete Cluster")
		return ctrl.Result{}, err
	}

	// OCICluster
	ociCluster := &infrastructurev1beta2.OCICluster{
		ObjectMeta: metav1.ObjectMeta{
			Name:      instance.Name,
			Namespace: capiSystemNamespace,
		},
	}
	if err := r.Delete(ctx, ociCluster); err != nil && !apierrors.IsNotFound(err) {
		log.Error(err, "Failed to delete OCICluster")
		return ctrl.Result{}, err
	}

	// Delete RBAC resources
	role := &rbacv1.ClusterRole{
		ObjectMeta: metav1.ObjectMeta{
			Name: "oci-cluster-autoscaler-extra",
		},
	}
	if err := r.Delete(ctx, role); err != nil && !apierrors.IsNotFound(err) {
		log.Error(err, "Failed to delete ClusterRole")
		return ctrl.Result{}, err
	}

	binding := &rbacv1.ClusterRoleBinding{
		ObjectMeta: metav1.ObjectMeta{
			Name: "oci-cluster-autoscaler-extra",
		},
	}
	if err := r.Delete(ctx, binding); err != nil && !apierrors.IsNotFound(err) {
		log.Error(err, "Failed to delete ClusterRoleBinding")
		return ctrl.Result{}, err
	}

	scc := &securityv1.SecurityContextConstraints{
		ObjectMeta: metav1.ObjectMeta{
			Name: "oci-capi",
		},
	}
	if err := r.Delete(ctx, scc); err != nil && !apierrors.IsNotFound(err) {
		log.Error(err, "Failed to delete SecurityContextConstraints")
		return ctrl.Result{}, err
	}

	// Remove finalizer
	controllerutil.RemoveFinalizer(instance, "ocicapicluster.openshift.io/finalizer")
	if err := r.Client.Update(ctx, instance); err != nil {
		return ctrl.Result{}, err
	}

	return ctrl.Result{}, nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *OCICAPIClusterReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&ocicapiv1alpha1.OCICAPICluster{}).
		Complete(r)
}
