package main

import (
	"context"
	"log"

	"github.com/ValentinGerlach/omink/pkg/env"
	"github.com/ValentinGerlach/omink/pkg/steps"
	"github.com/ValentinGerlach/omink/pkg/steps/kind"
	"github.com/ValentinGerlach/omink/pkg/steps/openmcp"
	"sigs.k8s.io/kind/pkg/apis/config/v1alpha4"
)

const (
	operatorVersion = "v0.17.1"
	operatorImage   = "ghcr.io/openmcp-project/images/openmcp-operator:" + operatorVersion
)

func main() {
	e := env.NewEnvironment("dev")
	ctx := context.Background()
	ctx = e.AddToContext(ctx)

	platformClusterName := e.GenerateKindClusterName("platform")

	s := []steps.Step{}
	s = append(s, kind.CreateCluster(&v1alpha4.Cluster{
		Name: platformClusterName,
		Nodes: []v1alpha4.Node{
			{
				Role: v1alpha4.ControlPlaneRole,
				ExtraMounts: []v1alpha4.Mount{
					{
						HostPath:      "/var/run/docker.sock",
						ContainerPath: "/var/run/host-docker.sock",
					},
				},
			},
		},
	})...)
	s = append(s, openmcp.DeployOperator(operatorImage, platformClusterName)...)

	if err := steps.Run(ctx, s); err != nil {
		log.Fatalln(err)
	}

	// *** Steps ***
	// Create platform cluster
	// Load images
	// Create openmcp-system Namespace
	// Create openmcp-operator ServiceAccount
	// Create ClusterRoleBinding for openmcp-operator
	// Create ConfigMap for openmcp-operator
	// Create openmcp-operator Deployment
	// Pre: Wait for ClusterProvider CRD to be created
	// Install ClusterProvider for kind
	// Create Cluster resource for platform
	// Install Service Provider Crossplane
	// Pre: Waiting for Crossplane ProviderConfig CRD to be available
	// Apply ProviderConfig
	// Waiting for the onboarding cluster to be created
	// Waiting for the onboarding cluster to be ready
	// Export onboarding kubeconfig
	// Create ManagedControlPlane
	// Create Crossplane resource
}
