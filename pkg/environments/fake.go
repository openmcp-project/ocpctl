package environments

import "sigs.k8s.io/controller-runtime/pkg/client"

type fakeProvider struct {
	ensureCreated    bool
	ensureErr        error
	ensureCalledWith string
	client           client.Client
	clientErr        error
	listClusters     []string
	listErr          error
	deleteErr        error
	deletedClusters  []string
}

func (f *fakeProvider) EnsurePlatformCluster(name string) (bool, error) {
	f.ensureCalledWith = name
	return f.ensureCreated, f.ensureErr
}

func (f *fakeProvider) PlatformClusterClient(_ string) (client.Client, error) {
	return f.client, f.clientErr
}

func (f *fakeProvider) ListClusters() ([]string, error) {
	return f.listClusters, f.listErr
}

func (f *fakeProvider) DeleteCluster(name string) error {
	if f.deleteErr != nil {
		return f.deleteErr
	}
	f.deletedClusters = append(f.deletedClusters, name)
	return nil
}
