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
	"crypto/x509"
	"encoding/pem"
	"fmt"
	"strings"
	"time"

	capiv1alpha1 "github.com/openshift/oci-capi-operator/api/v1alpha1"
	certificatesv1 "k8s.io/api/certificates/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"
)

// CertificateApprovalReconciler reconciles CertificateSigningRequests for OCI machines
type CertificateApprovalReconciler struct {
	client.Client
	Scheme *runtime.Scheme
}

// +kubebuilder:rbac:groups=certificates.k8s.io,resources=certificatesigningrequests,verbs=get;list;watch;update
// +kubebuilder:rbac:groups=infrastructure.cluster.x-k8s.io,resources=ocimachines,verbs=get;list;watch

// Reconcile handles certificate approval for OCI machines
func (r *CertificateApprovalReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	logger := log.FromContext(ctx)

	// Fetch the CSR
	csr := &certificatesv1.CertificateSigningRequest{}
	err := r.Get(ctx, req.NamespacedName, csr)
	if err != nil {
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}

	// Skip if already approved or denied
	if isCSRApproved(csr) || isCSRDenied(csr) {
		return ctrl.Result{}, nil
	}

	// Only process CSRs for kubelet certificates
	if !isKubeletCSR(csr) {
		return ctrl.Result{}, nil
	}

	// Extract hostname from CSR
	hostname, err := getCSRHostname(csr)
	if err != nil {
		logger.Error(err, "Failed to extract hostname from CSR", "csr", csr.Name, "signerName", csr.Spec.SignerName)
		return ctrl.Result{}, nil
	}

	if hostname == "" {
		logger.V(1).Info("No hostname found in CSR", "csr", csr.Name, "signerName", csr.Spec.SignerName)
		return ctrl.Result{}, nil
	}

	// Check if there's a matching OCIMachine
	if r.hasMatchingOCIMachine(ctx, hostname) {
		logger.Info("Approving certificate for OCI machine", "csr", csr.Name, "hostname", hostname)

		// Approve the CSR
		now := metav1.NewTime(time.Now())
		csr.Status.Conditions = append(csr.Status.Conditions, certificatesv1.CertificateSigningRequestCondition{
			Type:               certificatesv1.CertificateApproved,
			Status:             corev1.ConditionTrue,
			Reason:             "OCIMachineApproval",
			Message:            "Approved by OCI CAPI operator for matching OCIMachine",
			LastUpdateTime:     now,
			LastTransitionTime: now,
		})

		if err := r.Status().Update(ctx, csr); err != nil {
			return ctrl.Result{}, err
		}
	}

	return ctrl.Result{}, nil
}

func isKubeletCSR(csr *certificatesv1.CertificateSigningRequest) bool {
	return csr.Spec.SignerName == "kubernetes.io/kube-apiserver-client-kubelet" ||
		csr.Spec.SignerName == "kubernetes.io/kubelet-serving"
}

func (r *CertificateApprovalReconciler) hasMatchingOCIMachine(ctx context.Context, hostname string) bool {
	logger := log.FromContext(ctx)

	// List OCIMachines to find a match
	machineList := &metav1.PartialObjectMetadataList{}
	machineList.SetGroupVersionKind(schema.GroupVersionKind{
		Group:   "infrastructure.cluster.x-k8s.io",
		Version: "v1beta2",
		Kind:    "OCIMachine",
	})

	err := r.List(ctx, machineList, client.InNamespace("capi-system"))
	if err != nil {
		logger.Error(err, "Failed to list OCIMachines")
		return false
	}

	for _, machine := range machineList.Items {
		// More precise matching - look for hostname as prefix or suffix of machine name
		// This handles cases like hostname="worker-1" and machine.Name="my-cluster-worker-1"
		if strings.Contains(machine.Name, hostname) || strings.Contains(hostname, machine.Name) {
			logger.V(1).Info("Found matching OCIMachine", "hostname", hostname, "machine", machine.Name)
			return true
		}
	}

	logger.V(1).Info("No matching OCIMachine found", "hostname", hostname, "machineCount", len(machineList.Items))
	return false
}

func getCSRHostname(csr *certificatesv1.CertificateSigningRequest) (string, error) {
	switch csr.Spec.SignerName {
	case "kubernetes.io/kube-apiserver-client-kubelet":
		return getHostnameFromClientKubeletCSR(csr)
	case "kubernetes.io/kubelet-serving":
		return getHostnameFromServingCSR(csr)
	default:
		return "", nil
	}
}

func getHostnameFromClientKubeletCSR(csr *certificatesv1.CertificateSigningRequest) (string, error) {
	if len(csr.Spec.Request) == 0 {
		return "", fmt.Errorf("CSR request is empty")
	}

	block, _ := pem.Decode(csr.Spec.Request)
	if block == nil || block.Type != "CERTIFICATE REQUEST" {
		return "", fmt.Errorf("failed to decode PEM block or invalid block type")
	}

	certReq, err := x509.ParseCertificateRequest(block.Bytes)
	if err != nil {
		return "", fmt.Errorf("failed to parse certificate request: %w", err)
	}

	// Extract CN from subject
	subject := certReq.Subject.CommonName
	if subject == "" {
		return "", fmt.Errorf("certificate request has empty CommonName")
	}

	if strings.HasPrefix(subject, "system:node:") {
		hostname := strings.TrimPrefix(subject, "system:node:")
		// Remove domain suffix if present
		if dotIndex := strings.Index(hostname, "."); dotIndex > 0 {
			hostname = hostname[:dotIndex]
		}
		if hostname == "" {
			return "", fmt.Errorf("extracted hostname is empty")
		}
		return hostname, nil
	}

	return "", fmt.Errorf("CommonName %q does not have expected system:node: prefix", subject)
}

func getHostnameFromServingCSR(csr *certificatesv1.CertificateSigningRequest) (string, error) {
	// For kubelet serving CSRs, extract from username
	username := csr.Spec.Username
	if username == "" {
		return "", fmt.Errorf("CSR username is empty")
	}

	// The username should be in the format system:node:hostname
	if strings.HasPrefix(username, "system:node:") {
		hostname := strings.TrimPrefix(username, "system:node:")
		if dotIndex := strings.Index(hostname, "."); dotIndex > 0 {
			hostname = hostname[:dotIndex]
		}
		if hostname == "" {
			return "", fmt.Errorf("extracted hostname is empty")
		}
		return hostname, nil
	}

	return "", fmt.Errorf("username %q does not have expected system:node: prefix", username)
}

func isCSRApproved(csr *certificatesv1.CertificateSigningRequest) bool {
	for _, condition := range csr.Status.Conditions {
		if condition.Type == certificatesv1.CertificateApproved {
			return true
		}
	}
	return false
}

func isCSRDenied(csr *certificatesv1.CertificateSigningRequest) bool {
	for _, condition := range csr.Status.Conditions {
		if condition.Type == certificatesv1.CertificateDenied {
			return true
		}
	}
	return false
}

// SetupWithManager sets up the controller with the Manager.
func (r *CertificateApprovalReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&certificatesv1.CertificateSigningRequest{}).
		Complete(r)
}

// deployCertificateApproval creates a deployment that handles certificate approval
func (r *OCIClusterAutoscalerReconciler) deployCertificateApproval(ctx context.Context, autoscaler *capiv1alpha1.OCIClusterAutoscaler) error {
	// The certificate approval is now handled by the CertificateApprovalReconciler
	// This function can be used to deploy additional certificate approval logic if needed
	return nil
}
