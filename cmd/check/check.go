package check

import (
	"errors"
	"flag"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/hashicorp/go-multierror"
	version "github.com/hashicorp/go-version"
	"github.com/hashicorp/tf-sdk-migrator/util"
	"github.com/mitchellh/cli"
)

const (
	CommandName             = "check"
	goVersionConstraint     = ">=1.12"
	sdkVersionConstraint    = ">=0.12.6"
	terraformDependencyPath = "github.com/hashicorp/terraform"
)

type command struct{}

func CommandFactory() (cli.Command, error) {
	return &command{}, nil
}

func (c *command) Help() string {
	return `Usage: tf-sdk-migrator check [--help] [--csv] [PATH]

  Checks whether the Terraform provider at PATH is ready to be migrated to the
  new Terraform provider SDK (v1).

  PATH is resolved relative to $GOPATH/src/. If PATH is not supplied, it is assumed
  that the current working directory contains a Terraform provider.

  By default, outputs a human-readable report and exits 0 if the provider is
  ready for migration, 1 otherwise.

Options:
  --csv    Output results in CSV format.

Example:
  tf-sdk-migrator check github.com/terraform-providers/terraform-provider-local
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

	var providerRepoName string
	var providerPath string
	if flags.NArg() == 1 {
		var err error
		providerRepoName := flags.Args()[0]
		providerPath, err = util.GetProviderPath(providerRepoName)
		if err != nil {
			log.Printf("Error finding provider %s: %s", providerRepoName, err)
			return 1
		}
	} else if flags.NArg() == 0 {
		var err error
		providerPath, err = os.Getwd()
		if err != nil {
			log.Printf("Error finding current working directory: %s", err)
			return 1
		}
	} else {
		return cli.RunResultHelp
	}

	ui := &cli.ColoredUi{
		OutputColor: cli.UiColorBlue,
		InfoColor:   cli.UiColorGreen,
		ErrorColor:  cli.UiColorRed,
		WarnColor:   cli.UiColorYellow,
		Ui: &cli.BasicUi{
			Reader:      os.Stdin,
			Writer:      os.Stdout,
			ErrorWriter: os.Stderr,
		},
	}
	if !csv {
		ui.Output("Checking Go runtime version ...")
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
	sdkVersion, sdkVersionSatisfiesConstraint, err := CheckProviderSDKVersion(providerPath)
	if !csv {
		if sdkVersionSatisfiesConstraint {
			ui.Info(fmt.Sprintf("SDK version %s: OK.", sdkVersion))
		} else if sdkVersion != "" {
			ui.Warn(fmt.Sprintf("SDK version does not satisfy constraint %s. Found SDK version: %s", sdkVersionConstraint, sdkVersion))
		} else {

			ui.Warn(fmt.Sprintf("SDK version could not be determined. Provider must use hashicorp/terraform SDK."))
		}
	}
	if err != nil {
		log.Printf("[WARN] Error getting SDK version for provider %s: %s", providerPath, err)
		return 1
	}

	if !csv {
		ui.Output("Checking whether provider uses deprecated SDK packages or identifiers...")
	}
	removedPackagesInUse, removedIdentsInUse, doesNotUseRemovedPackagesOrIdents, err := CheckSDKPackageImportsAndRefs(providerPath)
	if !csv {
		if err != nil {
			log.Printf("[WARN] Error determining use of deprecated SDK packages and identifiers: %s", err)
			return 1
		}
		if doesNotUseRemovedPackagesOrIdents {
			ui.Info("No imports of deprecated SDK packages or identifiers: OK.")
		} else {
			ui.Warn(fmt.Sprintf("Deprecated SDK packages in use: %+v", removedPackagesInUse))
			ui.Warn(fmt.Sprintf("Deprecated SDK identifiers in use: %+v", removedIdentsInUse))
		}
	}
	allConstraintsSatisfied := goVersionSatisfiesConstraint && providerUsesGoModules && sdkVersionSatisfiesConstraint && doesNotUseRemovedPackagesOrIdents
	if csv {
		ui.Output(fmt.Sprintf("go_version,go_version_satisfies_constraint,uses_go_modules,sdk_version,sdk_version_satisfies_constraint,does_not_use_removed_packages,all_constraints_satisfied\n%s,%t,%t,%s,%t,%t,%t", goVersion, goVersionSatisfiesConstraint, providerUsesGoModules, sdkVersion, sdkVersionSatisfiesConstraint, doesNotUseRemovedPackagesOrIdents, allConstraintsSatisfied))
	} else {
		var prettyProviderName string
		if providerRepoName != "" {
			prettyProviderName = " " + providerRepoName
		}
		if allConstraintsSatisfied {
			ui.Info(fmt.Sprintf("\nAll constraints satisfied. Provider%s can be migrated to the new SDK.\n", prettyProviderName))
			return 0
		} else if providerUsesGoModules && sdkVersionSatisfiesConstraint && doesNotUseRemovedPackagesOrIdents {
			ui.Info(fmt.Sprintf("\nProvider%s can be migrated to the new SDK, but Go version %s is recommended.\n", prettyProviderName, goVersionConstraint))
			return 0
		}

		ui.Warn("\nSome constraints not satisfied. Please resolve these before migrating to the new SDK.")
	}

	return 1
}

func CheckGoVersion(providerPath string) (goVersion string, satisfiesConstraint bool) {
	c, err := version.NewConstraint(goVersionConstraint)

	runtimeVersion := strings.TrimLeft(runtime.Version(), "go")
	v, err := version.NewVersion(runtimeVersion)
	if err != nil {
		log.Printf("[ERROR] Could not parse Go version %s", runtimeVersion)
		return "", false
	}

	return runtimeVersion, c.Check(v)
}

func CheckForGoModules(providerPath string) (usingModules bool) {
	if _, err := os.Stat(filepath.Join(providerPath, "/go.mod")); err != nil {
		log.Printf("[WARN] 'go.mod' file not found - provider %s is not using Go modules", providerPath)
		return false
	}
	return true
}

// since use of Go modules is necessary for SDKv1 upgrade eligibility,
// we only run this check if the Go modules check has already passed
func CheckProviderSDKVersion(providerPath string) (sdkVersion string, satisfiesConstraint bool, error error) {
	c, err := version.NewConstraint(sdkVersionConstraint)

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
