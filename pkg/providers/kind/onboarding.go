package kind

const onboardingClusterSuffix = "-onboarding"

func OnboardingClusterName(environment string) string {
	return environment + onboardingClusterSuffix
}
