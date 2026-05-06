package clusters

const onboardingClusterSuffix = "-onboarding"

func OnboardingClusterName(environment string) string {
	return environment + onboardingClusterSuffix
}
