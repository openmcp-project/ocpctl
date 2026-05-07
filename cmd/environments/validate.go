package environments

import (
	"fmt"
	"regexp"

	"github.com/spf13/cobra"
)

const maxEnvironmentNameLength = 36

var validEnvironmentName = regexp.MustCompile(`^[a-z0-9.-]+$`)

func validateEnvironmentName(name string) error {
	if len(name) > maxEnvironmentNameLength {
		return fmt.Errorf("environment name %q must not exceed %d characters", name, maxEnvironmentNameLength)
	}
	if !validEnvironmentName.MatchString(name) {
		return fmt.Errorf("environment name %q is invalid, must match %s", name, validEnvironmentName)
	}
	return nil
}

func validatedNameArg(cmd *cobra.Command, args []string) error {
	if err := cobra.ExactArgs(1)(cmd, args); err != nil {
		return err
	}
	return validateEnvironmentName(args[0])
}
