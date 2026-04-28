package platform

import (
	"context"

	"github.com/openmcp-project/ocpctl/pkg/resources"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// OperatorNamespace returns a Resource for the given namespace.
func OperatorNamespace(namespace string) *resources.Resource {
	ns := &corev1.Namespace{
		ObjectMeta: metav1.ObjectMeta{
			Name: namespace,
		},
	}

	return &resources.Resource{
		Object: ns,
		ReadyFn: func(ctx context.Context) (bool, error) {
			return ns.Status.Phase == corev1.NamespaceActive, nil
		},
	}
}
