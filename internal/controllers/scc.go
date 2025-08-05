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

	securityv1 "github.com/openshift/api/security/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"

	capiv1alpha1 "github.com/openshift/oci-capi-operator/api/v1alpha1"
)

// createSecurityContextConstraints creates the SCC for the OCIClusterAutoscaler
// This is for the CAPI manager and CAPOCI controller manager
func (r *OCIClusterAutoscalerReconciler) createSecurityContextConstraints(ctx context.Context, autoscaler *capiv1alpha1.OCIClusterAutoscaler) error {
	sccName := "oci-capi"

	scc := &securityv1.SecurityContextConstraints{
		ObjectMeta: metav1.ObjectMeta{
			Name: sccName,
		},
	}

	_, err := controllerutil.CreateOrUpdate(ctx, r.Client, scc, func() error {
		// Set the desired state of the SCC
		scc.RunAsUser = securityv1.RunAsUserStrategyOptions{
			Type: securityv1.RunAsUserStrategyRunAsAny,
		}
		scc.SELinuxContext = securityv1.SELinuxContextStrategyOptions{
			Type: securityv1.SELinuxStrategyRunAsAny,
		}
		scc.SeccompProfiles = []string{"runtime/default"}
		scc.Users = []string{
			"system:serviceaccount:cluster-api-provider-oci-system:capoci-controller-manager",
			"system:serviceaccount:capi-system:capi-manager",
		}
		return nil
	})

	return err
}
