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

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"

	capiv1alpha1 "github.com/openshift/oci-capi-operator/api/v1alpha1"
)

// SecurityContextConstraints represents the OpenShift SCC
type SecurityContextConstraints struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	AllowHostDirVolumePlugin bool               `json:"allowHostDirVolumePlugin,omitempty"`
	AllowHostIPC             bool               `json:"allowHostIPC,omitempty"`
	AllowHostNetwork         bool               `json:"allowHostNetwork,omitempty"`
	AllowHostPID             bool               `json:"allowHostPID,omitempty"`
	AllowHostPorts           bool               `json:"allowHostPorts,omitempty"`
	AllowPrivilegedContainer bool               `json:"allowPrivilegedContainer,omitempty"`
	AllowedCapabilities      []string           `json:"allowedCapabilities,omitempty"`
	DefaultAddCapabilities   []string           `json:"defaultAddCapabilities,omitempty"`
	FSGroup                  FSGroup            `json:"fsGroup,omitempty"`
	ReadOnlyRootFilesystem   bool               `json:"readOnlyRootFilesystem,omitempty"`
	RequiredDropCapabilities []string           `json:"requiredDropCapabilities,omitempty"`
	RunAsUser                RunAsUser          `json:"runAsUser,omitempty"`
	SELinuxContext           SELinux            `json:"seLinuxContext,omitempty"`
	SeccompProfiles          []string           `json:"seccompProfiles,omitempty"`
	SupplementalGroups       SupplementalGroups `json:"supplementalGroups,omitempty"`
	Users                    []string           `json:"users,omitempty"`
	Groups                   []string           `json:"groups,omitempty"`
	Volumes                  []string           `json:"volumes,omitempty"`
}

type FSGroup struct {
	Type string `json:"type,omitempty"`
}

type RunAsUser struct {
	Type string `json:"type,omitempty"`
}

type SELinux struct {
	Type string `json:"type,omitempty"`
}

type SupplementalGroups struct {
	Type string `json:"type,omitempty"`
}

func (r *OCIClusterAutoscalerReconciler) createSecurityContextConstraints(ctx context.Context, autoscaler *capiv1alpha1.OCIClusterAutoscaler) error {
	sccName := "oci-capi"

	// Create the SCC using unstructured since we don't have the OpenShift types
	u := &unstructured.Unstructured{}
	u.SetAPIVersion("security.openshift.io/v1")
	u.SetKind("SecurityContextConstraints")
	u.SetName(sccName)

	_, err := controllerutil.CreateOrUpdate(ctx, r.Client, u, func() error {
		// Set the desired state of the SCC
		u.Object["runAsUser"] = map[string]interface{}{
			"type": "RunAsAny",
		}
		u.Object["seLinuxContext"] = map[string]interface{}{
			"type": "RunAsAny",
		}
		u.Object["seccompProfiles"] = []string{"runtime/default"}
		u.Object["users"] = []string{
			"system:serviceaccount:cluster-api-provider-oci-system:capoci-controller-manager",
			"system:serviceaccount:capi-system:capi-manager",
		}
		u.Object["volumes"] = []string{
			"configMap",
			"downwardAPI",
			"emptyDir",
			"persistentVolumeClaim",
			"projected",
			"secret",
		}
		return nil
	})

	return err
}
