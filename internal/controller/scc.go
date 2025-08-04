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

package controller

import (
	"context"

	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"

	capiv1beta1 "github.com/openshift/oci-capi-operator/api/v1beta1"
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

func (r *OCIClusterAutoscalerReconciler) createSecurityContextConstraints(ctx context.Context, autoscaler *capiv1beta1.OCIClusterAutoscaler) error {
	sccName := "oci-capi"

	// Check if SCC already exists
	existing := &SecurityContextConstraints{}
	err := r.Get(ctx, types.NamespacedName{Name: sccName}, existing)
	if err == nil {
		// SCC already exists
		return nil
	}
	if !errors.IsNotFound(err) {
		return err
	}

	// Create the SCC
	scc := &SecurityContextConstraints{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "security.openshift.io/v1",
			Kind:       "SecurityContextConstraints",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name: sccName,
		},
		RunAsUser: RunAsUser{
			Type: "RunAsAny",
		},
		SELinuxContext: SELinux{
			Type: "RunAsAny",
		},
		SeccompProfiles: []string{"runtime/default"},
		Users: []string{
			"system:serviceaccount:cluster-api-provider-oci-system:capoci-controller-manager",
			"system:serviceaccount:capi-system:capi-manager",
		},
		Volumes: []string{
			"configMap",
			"downwardAPI",
			"emptyDir",
			"persistentVolumeClaim",
			"projected",
			"secret",
		},
	}

	// Create the SCC using unstructured since we don't have the OpenShift types
	obj := &runtime.UnstructuredConverter{}
	unstructured, err := obj.ToUnstructured(scc)
	if err != nil {
		return err
	}

	u := &client.Object{}
	*u = &client.Object{}
	(*u).SetUnstructuredContent(unstructured)

	return r.Create(ctx, *u)
}
