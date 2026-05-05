package clusters

const OnboardingClusterSuffix = "-onboarding"

func OnboardingClusterName(environment string) string {
	return environment + OnboardingClusterSuffix
}
