package resources

import (
	"context"
	"fmt"
	"reflect"

	"github.com/openmcp-project/ocpctl/pkg/logging"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
)

// MutateFn sets the desired state on a resource object before it is created or updated.
type MutateFn func(ctx context.Context) error

// ReadyFn reports whether a resource has reached a ready state.
type ReadyFn func(ctx context.Context) (bool, error)

// Resource describes a single Kubernetes object to be reconciled, together with
// optional mutation and readiness logic and a list of objects it depends on.
// If External is true, the resource is never created or updated — it is only
// used as a dependency whose readiness can be checked via ReadyFn.
type Resource struct {
	Object       client.Object
	External     bool
	MutateFn     MutateFn
	Dependencies []client.Object
	ReadyFn      ReadyFn
}

// String returns a human-readable identifier in the form kind/namespace/name.
func (r *Resource) String() string {
	return objectString(r.Object)
}

// objectString formats a client.Object as kind/namespace/name.
func objectString(obj client.Object) string {
	kind := reflect.TypeOf(obj).Elem().Name()
	return fmt.Sprintf("%s/%s/%s", kind, obj.GetNamespace(), obj.GetName())
}

// ApplySummary holds the result of an Apply call.
type ApplySummary struct {
	Applied int
	Skipped int
}

// Cluster groups a controller-runtime client with the set of resources that
// should be reconciled against that cluster.
type Cluster struct {
	Client    client.Client
	Resources []*Resource
}

// AddResources appends resources to the cluster.
func (c *Cluster) AddResources(rs ...*Resource) {
	for _, r := range rs {
		c.Resources = append(c.Resources, r)
	}
}

// Manager coordinates applying resources across multiple clusters.
type Manager struct {
	Clusters []*Cluster
}

// AddClusters appends clusters to the manager.
func (m *Manager) AddClusters(cs ...*Cluster) {
	for _, c := range cs {
		m.Clusters = append(m.Clusters, c)
	}
}

// Apply creates or updates all resources across all clusters. Resources whose
// dependencies are not yet ready are skipped. Returns a summary of how many
// resources were applied and how many were skipped.
func (m *Manager) Apply(ctx context.Context) (ApplySummary, error) {
	index := make(map[client.Object]resourceEntry)
	for _, c := range m.Clusters {
		for _, r := range c.Resources {
			index[r.Object] = resourceEntry{resource: r, cluster: c}
		}
	}

	var total ApplySummary
	for _, c := range m.Clusters {
		s, err := applyCluster(ctx, c, index)
		if err != nil {
			return total, err
		}
		total.Applied += s.Applied
		total.Skipped += s.Skipped
	}
	return total, nil
}

type resourceEntry struct {
	resource *Resource
	cluster  *Cluster
}

// applyCluster creates or updates all resources in c, skipping those whose
// dependencies are not ready. index must cover all resources across all clusters
// so that cross-cluster dependencies can be resolved.
func applyCluster(ctx context.Context, c *Cluster, index map[client.Object]resourceEntry) (ApplySummary, error) {
	log := logging.FromContext(ctx)
	var summary ApplySummary
	for _, r := range c.Resources {
		if r.External {
			continue
		}
		ready, err := dependenciesReady(ctx, r, index)
		if err != nil {
			return summary, fmt.Errorf("checking dependencies for %s: %w", r, err)
		}
		if !ready {
			log.Debugf("Skipping %s: dependencies not ready", r)
			summary.Skipped++
			continue
		}

		mutateFn := controllerutil.MutateFn(func() error { return nil })
		if r.MutateFn != nil {
			mutateFn = func() error { return r.MutateFn(ctx) }
		}

		result, err := controllerutil.CreateOrUpdate(ctx, c.Client, r.Object, mutateFn)
		if err != nil {
			return summary, fmt.Errorf("applying %s: %w", r, err)
		}
		log.Debugf("Applied %s (%s)", r, result)
		summary.Applied++
	}
	return summary, nil
}

// dependenciesReady fetches the latest state of each dependency from its
// cluster and calls its ReadyFn. Returns false if any dependency is not found
// in the index or reports not ready. A dependency without a ReadyFn is assumed ready.
func dependenciesReady(ctx context.Context, r *Resource, index map[client.Object]resourceEntry) (bool, error) {
	log := logging.FromContext(ctx)
	for _, dep := range r.Dependencies {
		entry, ok := index[dep]
		if !ok {
			log.Debugf("Dependency %s of %s not found in index", objectString(dep), r)
			return false, nil
		}
		if err := entry.cluster.Client.Get(ctx, client.ObjectKeyFromObject(dep), dep); err != nil {
			return false, fmt.Errorf("getting dependency %s: %w", entry.resource, err)
		}
		if entry.resource.ReadyFn == nil {
			continue
		}
		ready, err := entry.resource.ReadyFn(ctx)
		if err != nil {
			return false, err
		}
		if !ready {
			log.Debugf("Dependency %s of %s is not ready", entry.resource, r)
			return false, nil
		}
	}
	return true, nil
}
