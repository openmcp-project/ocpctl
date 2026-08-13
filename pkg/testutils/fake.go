package testutils

import (
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// FakeProvider implements providers.ClusterProvider.
type FakeProvider struct {
	EnsureCreated    bool
	EnsureErr        error
	EnsureCalledWith string
	Client           client.Client
	ClientErr        error
	ListResult       []string
	ListErr          error
	DeleteErr        error
	DeletedClusters  []string
	KubeconfigData   string
	KubeconfigErr    error
	ExportedTo       string
	ExportErr        error
}

func (f *FakeProvider) EnsurePlatformCluster(name string) (bool, error) {
	f.EnsureCalledWith = name
	return f.EnsureCreated, f.EnsureErr
}

func (f *FakeProvider) PlatformClusterClient(_ string) (client.Client, error) {
	return f.Client, f.ClientErr
}

func (f *FakeProvider) ListClusters() ([]string, error) {
	return f.ListResult, f.ListErr
}

func (f *FakeProvider) DeleteCluster(name string) error {
	if f.DeleteErr != nil {
		return f.DeleteErr
	}
	f.DeletedClusters = append(f.DeletedClusters, name)
	return nil
}

func (f *FakeProvider) GetKubeconfig(_ string, _ bool) (string, error) {
	if f.KubeconfigErr != nil {
		return "", f.KubeconfigErr
	}
	return f.KubeconfigData, nil
}

func (f *FakeProvider) ExportKubeconfig(_ string, path string, _ bool) error {
	if f.ExportErr != nil {
		return f.ExportErr
	}
	f.ExportedTo = path
	return nil
}
