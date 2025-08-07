package autoscaler

import (
	capiv1alpha1 "github.com/openshift/oci-capi-operator/api/v1alpha1"
	ocicapiv1alpha1 "github.com/openshift/oci-capi-operator/api/v1alpha1"
	"github.com/openshift/oci-capi-operator/internal/components"

	appsv1 "k8s.io/api/apps/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
)

func NewComponent(capiSystemNamespace string, image string, autoscaler *capiv1alpha1.OCIClusterAutoscaler, scheme *runtime.Scheme) *components.Component {
	deploy, deployMutateFn := AutoscalerDeployment(capiSystemNamespace, image, scheme, autoscaler)
	serviceAccount, serviceAccountMutateFn := ServiceAccount(capiSystemNamespace, scheme, autoscaler)
	clusterRole, clusterRoleMutateFn := ClusterRole(capiSystemNamespace, autoscaler)
	clusterRoleBinding, clusterRoleBindingMutateFn := ClusterRoleBinding(capiSystemNamespace, autoscaler)

	return &components.Component{
		Name: "autoscaler",
		Subcomponents: components.SubcomponentList{
			{Name: "deployment", Object: deploy, MutateFn: deployMutateFn},
			{Name: "serviceAccount", Object: serviceAccount, MutateFn: serviceAccountMutateFn},
			{Name: "clusterRole", Object: clusterRole, MutateFn: clusterRoleMutateFn},
			{Name: "clusterRoleBinding", Object: clusterRoleBinding, MutateFn: clusterRoleBindingMutateFn},
		},
	}
}

func AutoscalerDeployment(capiSystemNamespace string, image string, scheme *runtime.Scheme, instance *ocicapiv1alpha1.OCIClusterAutoscaler) (client.Object, func() error) {
	deploy := &appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "oci-cluster-autoscaler",
			Namespace: "oci-cluster-autoscaler",
		},
	}

	mutateFn := func() error {
		// TODO: fill this in
		return controllerutil.SetControllerReference(instance, deploy, scheme)
	}

	return deploy, mutateFn
}
