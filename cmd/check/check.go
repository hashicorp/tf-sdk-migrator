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
	sdkVersionConstraint    = ">=0.12.7"
	terraformDependencyPath = "github.com/hashicorp/terraform"
)

type command struct {
	ui cli.Ui
}

func CommandFactory(ui cli.Ui) func() (cli.Command, error) {
	return func() (cli.Command, error) {
		return &command{ui}, nil
	}
}

func (c *command) Help() string {
	return `Usage: tf-sdk-migrator check [--help] [--csv] [IMPORT_PATH]

  Checks whether the Terraform provider at PATH is ready to be migrated to the
  new Terraform provider SDK (v1).

  IMPORT_PATH is resolved relative to $GOPATH/src/IMPORT_PATH. If it is not supplied,
  it is assumed that the current working directory contains a Terraform provider.

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

	if !csv {
		c.ui.Output("Checking Go runtime version ...")
	}
	goVersion, goVersionSatisfied := CheckGoVersion(providerPath)
	if !csv {
		if goVersionSatisfied {
			c.ui.Info(fmt.Sprintf("Go version %s: OK.", goVersion))
		} else {
			c.ui.Warn(fmt.Sprintf("Go version does not satisfy constraint %s. Found Go version: %s.", goVersionConstraint, goVersion))
		}
	}

	if !csv {
		c.ui.Output("Checking whether provider uses Go modules...")
	}
	goModulesUsed := CheckForGoModules(providerPath)
	if !csv {
		if goModulesUsed {
			c.ui.Info("Go modules in use: OK.")
		} else {
			c.ui.Warn("Go modules not in use. Provider must use Go modules.")
		}
	}

	if !csv {
		c.ui.Output("Checking version of github.com/hashicorp/terraform SDK used in provider...")
	}
	sdkVersion, sdkVersionSatisfied, err := CheckProviderSDKVersion(providerPath)
	if !csv {
		if sdkVersionSatisfied {
			c.ui.Info(fmt.Sprintf("SDK version %s: OK.", sdkVersion))
		} else if sdkVersion != "" {
			c.ui.Warn(fmt.Sprintf("SDK version does not satisfy constraint %s. Found SDK version: %s", sdkVersionConstraint, sdkVersion))
		} else {

			c.ui.Warn(fmt.Sprintf("SDK version could not be determined. Provider must use hashicorp/terraform SDK."))
		}
	}
	if err != nil {
		log.Printf("[WARN] Error getting SDK version for provider %s: %s", providerPath, err)
		return 1
	}

	if !csv {
		c.ui.Output("Checking whether provider uses deprecated SDK packages or identifiers...")
	}
	removedPackagesInUse, removedIdentsInUse, err := CheckSDKPackageImportsAndRefs(providerPath)
	if err != nil {
		c.ui.Error(err.Error())
		return 1
	}
	usesRemovedPackagesOrIdents := len(removedPackagesInUse) > 0 || len(removedIdentsInUse) > 0
	if !csv {
		if err != nil {
			log.Printf("[WARN] Error determining use of deprecated SDK packages and identifiers: %s", err)
			return 1
		}
		if !usesRemovedPackagesOrIdents {
			c.ui.Info("No imports of deprecated SDK packages or identifiers: OK.")
		}
		formatRemovedPackages(c.ui, removedPackagesInUse)
		formatRemovedIdents(c.ui, removedIdentsInUse)
	}
	constraintsSatisfied := goVersionSatisfied && goModulesUsed && sdkVersionSatisfied && !usesRemovedPackagesOrIdents
	if csv {
		c.ui.Output(fmt.Sprintf("go_version,go_version_satisfies_constraint,uses_go_modules,sdk_version,sdk_version_satisfies_constraint,does_not_use_removed_packages,all_constraints_satisfied\n%s,%t,%t,%s,%t,%t,%t",
			goVersion, goVersionSatisfied, goModulesUsed, sdkVersion, sdkVersionSatisfied, !usesRemovedPackagesOrIdents, constraintsSatisfied))
	} else {
		var prettyProviderName string
		if providerRepoName != "" {
			prettyProviderName = " " + providerRepoName
		}
		if constraintsSatisfied {
			c.ui.Info(fmt.Sprintf("\nAll constraints satisfied. Provider%s can be migrated to the new SDK.\n", prettyProviderName))
			return 0
		} else if goModulesUsed && sdkVersionSatisfied && !usesRemovedPackagesOrIdents {
			c.ui.Info(fmt.Sprintf("\nProvider%s can be migrated to the new SDK, but Go version %s is recommended.\n", prettyProviderName, goVersionConstraint))
			return 0
		}

		c.ui.Warn("\nSome constraints not satisfied. Please resolve these before migrating to the new SDK.")

	}

	return 1
}

func formatRemovedPackages(ui cli.Ui, removedPackagesInUse []string) {
	if len(removedPackagesInUse) == 0 {
		return
	}

	ui.Warn("Deprecated SDK packages in use:")
	for _, pkg := range removedPackagesInUse {
		ui.Warn(fmt.Sprintf(" * %s", pkg))
	}
}

func formatRemovedIdents(ui cli.Ui, removedIdentsInUse []*Offence) {
	if len(removedIdentsInUse) == 0 {
		return
	}
	ui.Warn("Deprecated SDK identifiers in use:")
	for _, ident := range removedIdentsInUse {
		d := ident.IdentDeprecation
		ui.Warn(fmt.Sprintf(" * %s (%s)", d.Identifier.Name, d.ImportPath))

		for _, pos := range ident.Positions {
			ui.Warn(fmt.Sprintf("   * %s", pos))
		}
	}
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

func CheckSDKPackageImportsAndRefs(providerPath string) (removedPackagesInUse []string, packageRefsOffences []*Offence, err error) {
	var providerImportDetails *ProviderImportDetails

	providerImportDetails, err = GoListPackageImports(providerPath)
	if err != nil {
		return nil, nil, err
	}

	removedPackagesInUse, err = CheckSDKPackageImports(providerImportDetails)
	if err != nil {
		return nil, nil, err
	}

	packageRefsOffences, err = CheckSDKPackageRefs(providerImportDetails)
	if err != nil {
		return nil, nil, err
	}

	return
}
