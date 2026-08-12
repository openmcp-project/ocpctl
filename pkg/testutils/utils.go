package testutils

import (
	"context"
	"testing"

	kindv1alpha1 "github.com/openmcp-project/cluster-provider-kind/api/v1alpha1"
	"github.com/openmcp-project/ocpctl/pkg/logging"
	clustersv1alpha1 "github.com/openmcp-project/openmcp-operator/api/clusters/v1alpha1"
	apimeta "k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
)

func Ctx(t testing.TB) context.Context {
	t.Helper()
	log, err := logging.NewLogger(false)
	if err != nil {
		t.Fatal(err)
	}
	return logging.IntoContext(context.Background(), log)
}

func ClusterWithKindName(t testing.TB, name, kindName string) *clustersv1alpha1.Cluster {
	t.Helper()
	cl := &clustersv1alpha1.Cluster{ObjectMeta: metav1.ObjectMeta{Name: name}}
	ps := kindv1alpha1.ClusterStatus{
		TypeMeta: metav1.TypeMeta{
			Kind:       "ClusterStatus",
			APIVersion: kindv1alpha1.SchemeGroupVersion.String(),
		},
		KindClusterName: kindName,
	}
	if err := cl.Status.SetProviderStatus(ps); err != nil {
		t.Fatal(err)
	}
	return cl
}

// NoMatchClient wraps a fake client and returns a NoKindMatchError for List calls.
type NoMatchClient struct {
	client.Client
}

func (n *NoMatchClient) List(_ context.Context, _ client.ObjectList, _ ...client.ListOption) error {
	return &apimeta.NoKindMatchError{GroupKind: schema.GroupKind{Group: "clusters.openmcp.cloud", Kind: "ClusterList"}}
}

// SchemeClient returns a fake client with the clusters v1alpha1 scheme registered.
func SchemeClient(t testing.TB, objs ...client.Object) client.Client {
	t.Helper()
	s := runtime.NewScheme()
	if err := clustersv1alpha1.AddToScheme(s); err != nil {
		t.Fatal(err)
	}
	b := fake.NewClientBuilder().WithScheme(s)
	if len(objs) > 0 {
		b = b.WithStatusSubresource(objs...).WithObjects(objs...)
	}
	return b.Build()
}
