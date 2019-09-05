package migrate

import (
	"bufio"
	"flag"
	"fmt"
	"go/parser"
	"go/printer"
	"go/token"
	"io/ioutil"
	"log"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/hashicorp/tf-sdk-migrator/cmd/check"
	"github.com/hashicorp/tf-sdk-migrator/util"
	"github.com/mitchellh/cli"
	"github.com/radeksimko/mod/modfile"
)

const (
	CommandName       = "migrate"
	oldSDKImportPath  = "github.com/hashicorp/terraform"
	newSDKImportPath  = "github.com/hashicorp/terraform-plugin-sdk"
	newSDKPackagePath = "github.com/hashicorp/terraform-plugin-sdk"
	defaultSDKVersion = "v0.0.1"
)

var printConfig = printer.Config{
	Mode:     printer.TabIndent | printer.UseSpaces,
	Tabwidth: 8,
}

type command struct{}

func CommandFactory() (cli.Command, error) {
	return &command{}, nil
}

func (c *command) Help() string {
	return `Usage: tf-sdk-migrator migrate [--help] [--sdk-version SDK_VERSION] [PATH]

  Migrates the Terraform provider at PATH to the new Terraform provider
  SDK, defaulting to version ` + defaultSDKVersion + `.

  PATH is resolved relative to $GOPATH/src/. If PATH is not supplied, it is assumed
  that the current working directory contains a Terraform provider.

  Optionally, an SDK_VERSION can be passed, which is parsed as a Go module
  release version. For example: v1.0.1, latest, master.

  Rewrites import paths and go.mod. No backup is made before files are
  overwritten.

Example:
  tf-sdk-migrator migrate --sdk-version master github.com/terraform-providers/terraform-provider-local`
}

func (c *command) Synopsis() string {
	return "Migrates a Terraform provider to the new SDK (v1)."
}

func (c *command) Run(args []string) int {
	flags := flag.NewFlagSet(CommandName, flag.ExitOnError)
	var sdkVersion string
	flags.StringVar(&sdkVersion, "sdk-version", defaultSDKVersion, "SDK version")
	flags.Parse(args)

	var providerRepoName string
	var providerPath string
	if flags.NArg() == 1 {
		var err error
		providerRepoName = flags.Args()[0]
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

	checkCmd, err := check.CommandFactory()
	if err != nil {
		ui.Error(fmt.Sprintf("Error running eligibility check: %s", err))
	}

	var checkArgs []string
	if providerRepoName != "" {
		checkArgs = []string{providerRepoName}
	}
	returnCode := checkCmd.Run(checkArgs)
	if returnCode != 0 {
		ui.Warn("Provider failed eligibility check for migration to the new SDK. Please see warnings above.")
		return 1
	}

	ui.Output("Rewriting provider go.mod file...")
	err = RewriteGoMod(providerPath, sdkVersion)
	if err != nil {
		ui.Error(fmt.Sprintf("Error rewriting go.mod file: %s", err))
		return 1
	}

	ui.Output("Rewriting SDK package imports...")
	err = filepath.Walk(providerPath, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if info.IsDir() && info.Name() == "vendor" {
			return filepath.SkipDir
		}
		if !info.IsDir() && strings.HasSuffix(info.Name(), ".go") {
			err := RewriteImportedPackageImports(path, oldSDKImportPath, newSDKImportPath)
			if err != nil {
				return err
			}
		}
		return nil
	})
	if err != nil {
		ui.Error(fmt.Sprintf("Error rewriting SDK imports: %s", err))
		return 1
	}

	ui.Info(fmt.Sprintf("Success! Provider %s is migrated to %s %s.", providerPath, newSDKPackagePath, sdkVersion))
	return 0
}

func RewriteGoMod(providerPath string, sdkVersion string) error {
	goModPath := providerPath + "/go.mod"

	input, err := ioutil.ReadFile(goModPath)
	if err != nil {
		return err
	}

	pf, err := modfile.Parse(goModPath, input, nil)
	if err != nil {
		return err
	}

	err = pf.DropRequire(oldSDKImportPath)
	if err != nil {
		return err
	}

	pf.AddNewRequire(newSDKPackagePath, sdkVersion, false)

	pf.Cleanup()
	formattedOutput, err := pf.Format()
	if err != nil {
		return err
	}

	err = ioutil.WriteFile(goModPath, formattedOutput, 0644)
	if err != nil {
		return err
	}

	return nil
}

func RewriteImportedPackageImports(filePath string, stringToReplace string, replacement string) error {
	// TODO: check file exists so ParseFile doesn't panic
	fset := token.NewFileSet()
	f, err := parser.ParseFile(fset, filePath, nil, parser.ParseComments)
	if err != nil {
		return err
	}

	for _, impSpec := range f.Imports {
		impPath, err := strconv.Unquote(impSpec.Path.Value)
		if err != nil {
			log.Print(err)
		}
		// prevent partial matches on package names
		if impPath == stringToReplace || strings.HasPrefix(impPath, stringToReplace+"/") {
			newImpPath := strings.Replace(impPath, stringToReplace, replacement, -1)
			impSpec.Path.Value = strconv.Quote(newImpPath)
		}
	}

	out, err := os.Create(filePath)
	if err != nil {
		return err
	}
	defer out.Close()
	w := bufio.NewWriter(out)
	if err := printConfig.Fprint(w, fset, f); err != nil {
		return err
	}
	if err := w.Flush(); err != nil {
		return err
	}

	return nil
}
