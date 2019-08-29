package check

import (
	"errors"
	"flag"
	"fmt"
	"log"
	"os"
	"path/filepath"

	"github.com/hashicorp/go-multierror"
	version "github.com/hashicorp/go-version"
	"github.com/hashicorp/tf-sdk-migrator/util"
	"github.com/mitchellh/cli"
)

const (
	CommandName             = "check"
	goVersionConstraint     = ">=1.12"
	SDKVersionConstraint    = ">=0.12.6"
	terraformDependencyPath = "github.com/hashicorp/terraform"
)

type command struct{}

func CommandFactory() (cli.Command, error) {
	return &command{}, nil
}

func (c *command) Help() string {
	return `Usage: tf-sdk-migrator check [--help] [--csv] PATH

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
	flags := flag.NewFlagSet(CommandName, flag.ExitOnError)
	var csv bool
	flags.BoolVar(&csv, "csv", false, "CSV output")
	flags.Parse(args)

	var providerPath string
	if len(args) > 0 {
		var err error
		providerRepoName := args[len(args)-1]
		providerPath, err = util.GetProviderPath(providerRepoName)
		if err != nil {
			log.Printf("Error finding provider %s: %s", providerRepoName, err)
			return 1
		}
	} else {
		var err error
		providerPath, err = os.Getwd()
		if err != nil {
			log.Printf("Error finding current working directory: %s", err)
			return 1
		}
	}

	ui := &cli.ColoredUi{
		OutputColor: cli.UiColorNone,
		InfoColor:   cli.UiColorBlue,
		ErrorColor:  cli.UiColorRed,
		WarnColor:   cli.UiColorYellow,
		Ui: &cli.BasicUi{
			Reader:      os.Stdin,
			Writer:      os.Stdout,
			ErrorWriter: os.Stderr,
		},
	}
	if !csv {
		ui.Output("Checking Go version used in provider...")
	}
	goVersion, goVersionSatisfiesConstraint := CheckGoVersion(providerPath)
	if !csv {
		if goVersionSatisfiesConstraint {
			ui.Info(fmt.Sprintf("Go version %s: OK.", goVersion))
		} else {
			ui.Warn(fmt.Sprintf("Go version does not satisfy constraint %s. Found Go version: %s.", goVersionConstraint, goVersion))
		}
	}

	if !csv {
		ui.Output("Checking whether provider uses Go modules...")
	}
	providerUsesGoModules := CheckForGoModules(providerPath)
	if !csv {
		if providerUsesGoModules {
			ui.Info("Go modules in use: OK.")
		} else {
			ui.Warn("Go modules not in use. Provider must use Go modules.")
		}
	}

	if !csv {
		ui.Output("Checking version of github.com/hashicorp/terraform SDK used in provider...")
	}
	SDKVersion, SDKVersionSatisfiesConstraint, err := CheckProviderSDKVersion(providerPath)
	if !csv {
		if SDKVersionSatisfiesConstraint {
			ui.Info(fmt.Sprintf("SDK version %s: OK.", SDKVersion))
		} else {
			ui.Warn(fmt.Sprintf("SDK version does not satisfy constraint %s. Found SDK version: %s", SDKVersionConstraint, SDKVersion))
		}
	}
	if err != nil {
		log.Printf("Error getting SDK version for provider %s: %s", providerPath, err)
		return 1
	}

	if !csv {
		ui.Output("Checking whether provider uses deprecated SDK packages or identifiers...")
	}
	removedPackagesInUse, removedIdentsInUse, doesNotUseRemovedPackagesOrIdents, err := CheckSDKPackageImportsAndRefs(providerPath)
	if !csv {
		if err != nil {
			log.Printf("Error determining use of deprecated SDK packages and identifiers: %s", err)
			return 1
		}
		if doesNotUseRemovedPackagesOrIdents {
			ui.Info("No imports of deprecated SDK packages or identifiers: OK.")
		} else {
			ui.Warn(fmt.Sprintf("Deprecated SDK packages in use: %+v", removedPackagesInUse))
			ui.Warn(fmt.Sprintf("Deprecated SDK identifiers in use: %+v", removedIdentsInUse))
		}
	}
	allConstraintsSatisfied := goVersionSatisfiesConstraint && providerUsesGoModules && SDKVersionSatisfiesConstraint && doesNotUseRemovedPackagesOrIdents
	if csv {
		ui.Output(fmt.Sprintf("go_version,go_version_satisfies_constraint,uses_go_modules,sdk_version,sdk_version_satisfies_constraint,does_not_use_removed_packages,all_constraints_satisfied\n%s,%t,%t,%s,%t,%t,%t", goVersion, goVersionSatisfiesConstraint, providerUsesGoModules, SDKVersion, SDKVersionSatisfiesConstraint, doesNotUseRemovedPackagesOrIdents, allConstraintsSatisfied))
	} else {
		if allConstraintsSatisfied {
			ui.Info(fmt.Sprintf("\nAll constraints satisfied. Provider %s can be migrated to the new SDK.", providerPath))
			return 0
		} else if providerUsesGoModules && SDKVersionSatisfiesConstraint && doesNotUseRemovedPackagesOrIdents {
			ui.Info(fmt.Sprintf("\nProvider %s can be migrated to the new SDK, but Go version %s is recommended.", providerPath, goVersionConstraint))
			return 0
		}

		ui.Warn("\nSome constraints not satisfied. Please resolve these before migrating to the new SDK.")
	}

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

func CheckSDKPackageImportsAndRefs(providerPath string) (removedPackagesInUse []string, removedIdentsInUse []string, doesNotUseRemovedPackagesOrIdents bool, e error) {
	providerImportDetails, err := GoListPackageImports(providerPath)
	if err != nil {
		return nil, nil, false, err
	}

	removedPackagesInUse, err = CheckSDKPackageImports(providerImportDetails)
	if err != nil {
		return nil, nil, false, err
	}

	packageRefsOffences, err := CheckSDKPackageRefs(providerImportDetails)
	if err != nil {
		return nil, nil, false, err
	}
	for _, o := range packageRefsOffences {
		removedIdentsInUse = append(removedIdentsInUse, fmt.Sprintf("Ident %v from package %v is used at %+v", o.IdentDeprecation.Identifier.Name, o.IdentDeprecation.ImportPath, o.Positions))
	}

	return removedPackagesInUse, removedIdentsInUse, len(removedPackagesInUse) == 0 && len(packageRefsOffences) == 0, nil
}
