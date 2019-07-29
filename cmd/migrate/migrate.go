package migrate

import (
	"bufio"
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
)

const (
	oldSDKImportPath  = "github.com/hashicorp/terraform"
	newSDKImportPath  = "github.com/hashicorp/terraform-plugin-sdk/sdk"
	newSDKPackagePath = "github.com/hashicorp/terraform-plugin-sdk"
	newSDKVersion     = "v0.0.1"
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
	return ""
}

func (c *command) Synopsis() string {
	return ""
}

func (c *command) Run(args []string) int {
	// TODO --dry-run flag

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
	log.Println(providerPath)

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
	returnCode := checkCmd.Run(args)
	if returnCode != 0 {
		ui.Warn("Provider failed eligibility check for migration to the new SDK. Please see warnings above.")
		return 1
	}

	ui.Output("Rewriting provider go.mod file...")
	err = RewriteGoMod(providerPath)
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

	ui.Info(fmt.Sprintf("Success! Provider %s is migrated to %s %s.", providerPath, newSDKPackagePath, newSDKVersion))
	return 0
}

func RewriteGoMod(providerPath string) error {
	goModPath := providerPath + "/go.mod"

	input, err := ioutil.ReadFile(goModPath)
	if err != nil {
		return err
	}

	lines := strings.Split(string(input), "\n")

	// TODO: case where there is only one package in go.mod
	for i, line := range lines {
		if strings.HasPrefix(line, "\t"+oldSDKImportPath+" ") {
			lines[i] = "\t" + newSDKPackagePath + " " + newSDKVersion
		}
	}
	output := strings.Join(lines, "\n")
	err = ioutil.WriteFile(goModPath, []byte(output), 0644)
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
