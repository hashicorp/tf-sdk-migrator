package check

import (
	"errors"
	"log"
	"os"
	"path/filepath"

	"github.com/hashicorp/go-multierror"
	version "github.com/hashicorp/go-version"
	"github.com/kmoe/tf-sdk-migrator/util"
	"github.com/mitchellh/cli"
)

const (
	goVersionConstraint     = ">=1.12"
	SDKVersionConstraint    = ">=0.12"
	terraformDependencyPath = "github.com/hashicorp/terraform"
)

type command struct{}

func CommandFactory() (cli.Command, error) {
	return &command{}, nil
}

func (c *command) Help() string {
	return "help me"
}

func (c *command) Synopsis() string {
	return "m'synopsis"
}

func (c *command) Run(args []string) int {
	var providerRepoName string
	if len(args) > 0 {
		providerRepoName = args[0]
	}
	providerPath, err := util.GetProviderPath(providerRepoName)
	if err != nil {
		log.Printf("Error finding provider %s: %s", providerRepoName, err)
		return 1
	}

	// perform all checks, then display results
	// for each check, there is an output value
	// and a bool saying whether the check passed
	goVersion, goVersionSatisfiesConstraint := CheckGoVersion(providerPath)

	// check for go 1.12+

	// check that all dependencies are tracked via go modules
	providerUsesGoModules := CheckForGoModules(providerPath)

	SDKVersionSatisfiesConstraint := false
	SDKVersion := ""

	// check that provider uses latest SDK v 0.12.x
	// TODO: simplify by only checking if provider does use modules?
	if providerUsesGoModules {
		SDKVersion, SDKVersionSatisfiesConstraint, err = CheckProviderSDKVersion(providerPath)
		if err != nil {
			log.Printf("Error getting SDK version for provider %s: %s", providerPath, err)
			return 1
		}
	}

	log.Printf("Go Version: " + goVersion + " SDK Version: " + SDKVersion)

	// check that provider doesn't use packages we're removing

	if goVersionSatisfiesConstraint && providerUsesGoModules && SDKVersionSatisfiesConstraint {
		log.Printf("all constraints satisfied!")
		return 0
	}

	log.Printf("some constraints not satisfied!")

	return 1
}

func CheckGoVersion(providerPath string) (goVersion string, satisfiesConstraint bool) {
	c, err := version.NewConstraint(goVersionConstraint)

	v, err := ReadGoVersionFromGoVersionFile(providerPath)
	if err != nil {
		log.Printf("no Go version found in .go-version file for %s: %s", providerPath, err)
	} else if v != nil {
		return v.String(), c.Check(v)
	}

	v, err = ReadGoVersionFromGoModFile(providerPath)
	if err != nil {
		log.Printf("no go version found in go.mod file for %s: %s", providerPath, err)
	} else if v != nil {
		return v.String(), c.Check(v)
	}

	v, err = ReadGoVersionFromTravisConfig(providerPath)
	if err != nil {
		log.Printf("no go version found in Travis config file for %s: %s", providerPath, err)
	} else if v != nil {
		return v.String(), c.Check(v)
	}

	log.Printf("failed to detect Go version for provider %s", providerPath)

	return "", false
}

func CheckForGoModules(providerPath string) (usingModules bool) {
	if _, err := os.Stat(filepath.Join(providerPath, "/go.mod")); err != nil {
		log.Printf("'go.mod' file not found - provider %s is not using Go modules", providerPath)
		return false
	}
	return true
}

// since use of Go modules is necessary for SDKv1 upgrade eligibility,
// we only run this check if the Go modules check has already passed
func CheckProviderSDKVersion(providerPath string) (SDKVersion string, satisfiesConstraint bool, error error) {
	c, err := version.NewConstraint(SDKVersionConstraint)

	v, err := ReadSDKVersionFromGoModFile(providerPath)
	if err != nil {
		return "", false, multierror.Append(err, errors.New("could not read SDK version from go.mod file for provider"+providerPath))
	}

	return v.String(), c.Check(v), nil
}

// func CheckSDKPackageImports(providerPath string) (packageImports string, includesRemovedPackages bool) {
// 	return nil
// }
