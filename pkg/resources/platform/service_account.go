package platform

import (
	"github.com/openmcp-project/ocpctl/pkg/resources"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// OperatorServiceAccount returns a Resource for the openmcp-operator ServiceAccount.
// The namespace resource must be applied before this one.
func OperatorServiceAccount(ns *corev1.Namespace) *resources.Resource {
	sa := &corev1.ServiceAccount{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "openmcp-operator",
			Namespace: ns.Name,
		},
	}

	return &resources.Resource{
		Object:       sa,
		Dependencies: []client.Object{ns},
	}
}
