package check

import (
	"errors"
	"io/ioutil"
	"log"
	"os"
	"path/filepath"
	"strings"

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
	return `Usage: tf-sdk-migrator check [--help] PATH

  Checks whether the Terraform provider at PATH is ready to be migrated to the
  new Terraform provider SDK (v1).

  By default, outputs a human-readable report and exits 0 if the provider is
  ready for migration, 1 otherwise.

Options:
  ---csv    Output results in CSV format.
`
}

func (c *command) Synopsis() string {
	return "Checks whether a Terraform provider is ready to be migrated to the new SDK (v1)."
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

	goVersion, goVersionSatisfiesConstraint := CheckGoVersion(providerPath)

	providerUsesGoModules := CheckForGoModules(providerPath)

	SDKVersionSatisfiesConstraint := false
	SDKVersion := ""
	if providerUsesGoModules {
		SDKVersion, SDKVersionSatisfiesConstraint, err = CheckProviderSDKVersion(providerPath)
		if err != nil {
			log.Printf("Error getting SDK version for provider %s: %s", providerPath, err)
			return 1
		}
	}
	log.Printf("Go Version: " + goVersion + " SDK Version: " + SDKVersion)

	removedPackagesInUse, doesNotUseRemovedPackages, err := CheckSDKPackageImports(providerPath)
	if err != nil {
		log.Printf("Error determining use of removed SDK packages: %s", err)
		return 1
	}
	if doesNotUseRemovedPackages {
		log.Println("No use of removed SDK packages detected")
	} else {
		log.Printf("Removed SDK packages in use: %+v", removedPackagesInUse)
	}

	if goVersionSatisfiesConstraint && providerUsesGoModules && SDKVersionSatisfiesConstraint && doesNotUseRemovedPackages {
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

func CheckSDKPackageImports(providerPath string) (removedPackagesInUse []string, doesNotUseRemovedPackages bool, e error) {
	removedPackages, err := readRemovedPackagesFile("REMOVED_PACKAGES")
	if err != nil {
		return []string{}, false, err
	}

	removedPackagesInUse = []string{}

	filepath.Walk(providerPath, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if info.IsDir() && info.Name() == "vendor" {
			return filepath.SkipDir
		}
		if !info.IsDir() && strings.HasSuffix(info.Name(), ".go") {
			removedPackagesInUse = append(removedPackagesInUse, util.FindImportedPackages(path, removedPackages)...)
		}
		return nil
	})

	return removedPackagesInUse, len(removedPackagesInUse) == 0, nil
}

func readRemovedPackagesFile(path string) ([]string, error) {
	// it's a small file
	content, err := ioutil.ReadFile(path)
	if err != nil {
		return []string{}, err
	}
	lines := strings.Split(string(content), "\n")
	return lines, nil
}
