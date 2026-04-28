package version

// go build -ldflags "-X github.com/openmcp-project/ocpctl/internal/version.version=1.2.3"
var version = "dev"

func Version() string {
	return version
}
