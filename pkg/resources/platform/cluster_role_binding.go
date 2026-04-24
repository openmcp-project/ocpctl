package platform

import (
	"github.com/ValentinGerlach/oink/pkg/resources"
	corev1 "k8s.io/api/core/v1"
	rbacv1 "k8s.io/api/rbac/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// OperatorClusterRoleBinding returns a Resource that binds the openmcp-operator
// ServiceAccount to the cluster-admin ClusterRole. Depends on the service account.
func OperatorClusterRoleBinding(sa *corev1.ServiceAccount) *resources.Resource {
	crb := &rbacv1.ClusterRoleBinding{
		ObjectMeta: metav1.ObjectMeta{
			Name: "openmcp-operator",
		},
		RoleRef: rbacv1.RoleRef{
			APIGroup: rbacv1.GroupName,
			Kind:     "ClusterRole",
			Name:     "cluster-admin",
		},
		Subjects: []rbacv1.Subject{
			{
				Kind:      rbacv1.ServiceAccountKind,
				Name:      sa.Name,
				Namespace: sa.Namespace,
			},
		},
	}

	return &resources.Resource{
		Object:       crb,
		Dependencies: []client.Object{sa},
	}
}
