package versions

import "fmt"

type OciImage struct {
	Registry   string
	Repository string
	Tag        string
}

func (o OciImage) String() string {
	return fmt.Sprintf("%s/%s:%s", o.Registry, o.Repository, o.Tag)
}

func ClusterProviderKind() OciImage {
	return OciImage{
		Registry:   "ghcr.io",
		Repository: "openmcp-project/images/cluster-provider-kind",
		Tag:        "v0.2.0",
	}
}

func Operator() OciImage {
	return OciImage{
		Registry:   "ghcr.io",
		Repository: "openmcp-project/images/openmcp-operator",
		Tag:        "v0.18.1",
	}
}
