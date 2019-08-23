package check

import (
	"bytes"
	"encoding/json"
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"io"
	"os"
	"os/exec"
	"path"

	"github.com/hashicorp/tf-sdk-migrator/util"
	refsParser "github.com/radeksimko/go-refs/parser"
)

type Offence struct {
	IdentDeprecation *identDeprecation
	Positions        []*token.Position
}

type identDeprecation struct {
	ImportPath string
	Identifier *ast.Ident
	Message    string
}

var deprecations = []*identDeprecation{
	{
		"github.com/hashicorp/terraform/httpclient",
		ast.NewIdent("UserAgentString"),
		"Please don't use this",
	},
	{
		"github.com/hashicorp/terraform/terraform",
		ast.NewIdent("UserAgentString"),
		"Please don't use this",
	},
	{
		"github.com/hashicorp/terraform/terraform",
		ast.NewIdent("VersionString"),
		"Please don't use this",
	},
	{
		"github.com/hashicorp/terraform/config",
		ast.NewIdent("UserAgentString"),
		"Please don't use this",
	},
	{
		"github.com/hashicorp/terraform/config",
		ast.NewIdent("NewRawConfig"),
		"terraform.NewResourceConfig and config.NewRawConfig have been removed, please use terraform.NewResourceConfigRaw",
	},
	{
		"github.com/hashicorp/terraform/terraform",
		ast.NewIdent("NewResourceConfig"),
		"terraform.NewResourceConfig and config.NewRawConfig have been removed, please use terraform.NewResourceConfigRaw",
	},
}

// Package represents the subset of `go list` output we are interested in
type Package struct {
	Dir           string // directory containing package sources
	ImportPath    string // import path of package in dir
	ImportComment string // path in import comment on package statement

	// Source files
	GoFiles     []string // .go source files (excluding CgoFiles, TestGoFiles, XTestGoFiles)
	TestGoFiles []string // _test.go files in package

	// Dependency information
	Imports     []string          // import paths used by this package
	ImportMap   map[string]string // map from source import to ImportPath (identity entries omitted)
	Deps        []string          // all (recursively) imported dependencies
	TestImports []string          // imports from TestGoFiles

	// Error information
	Incomplete bool            // this package or a dependency has an error
	Error      *PackageError   // error loading package
	DepsErrors []*PackageError // errors loading dependencies
}

type PackageError struct {
	Err string
}

// ProviderImports is a data structure we parse the `go list` output into
// for efficient searching
type ProviderImportDetails struct {
	AllImportPathsHash map[string]bool
	Packages           map[string]ProviderPackage
}

type ProviderPackage struct {
	Dir         string
	ImportPath  string
	GoFiles     []string
	TestGoFiles []string
	Imports     []string
	TestImports []string
}

func GoCmd(workDir string, args ...string) (*bytes.Buffer, string, error) {
	cmd := exec.Command("go", args...)
	cmd.Dir = workDir

	var stdout, stderr bytes.Buffer
	cmd.Env = append(os.Environ(), "GO111MODULE=on")
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	err := cmd.Run()
	if err != nil {
		return nil, stderr.String(), fmt.Errorf("%q: %s", args, err)
	}

	return &stdout, stderr.String(), nil
}

func GoListPackageImports(providerPath string) (*ProviderImportDetails, error) {
	out, _, err := GoCmd(providerPath, "list", "-json", "./...")
	if err != nil {
		return nil, err
	}

	allImportPathsHash := make(map[string]bool)
	providerPackages := make(map[string]ProviderPackage)

	dec := json.NewDecoder(bytes.NewReader(out.Bytes()))
	for {
		var p Package
		if err := dec.Decode(&p); err != nil {
			if err == io.EOF {
				break
			}
			return nil, err
		}

		for _, i := range p.Imports {
			allImportPathsHash[i] = true
		}

		providerPackages[p.ImportPath] = ProviderPackage{
			Dir:         p.Dir,
			ImportPath:  p.ImportPath,
			GoFiles:     p.GoFiles,
			TestGoFiles: p.TestGoFiles,
			Imports:     p.Imports,
			TestImports: p.TestImports,
		}
	}

	return &ProviderImportDetails{
		AllImportPathsHash: allImportPathsHash,
		Packages:           providerPackages,
	}, nil
}

func CheckSDKPackageRefs(providerImportDetails *ProviderImportDetails) ([]*Offence, error) {
	offences := make([]*Offence, 0, 0)

	for _, d := range deprecations {
		fset := token.NewFileSet()
		files, err := filesWhichImport(providerImportDetails, d.ImportPath)
		if err != nil {
			return nil, err
		}

		foundPositions := make([]*token.Position, 0, 0)

		for _, filePath := range files {
			f, err := parser.ParseFile(fset, filePath, nil, 0)
			if err != nil {
				return nil, err
			}

			identifiers, err := refsParser.FindPackageReferences(f, d.ImportPath)
			if err != nil {
				// package not imported in this file
				continue
			}

			positions, err := findIdentifierPositions(fset, identifiers, d.Identifier)
			if err != nil {
				return nil, err
			}

			if len(positions) > 0 {
				foundPositions = append(foundPositions, positions...)
			}
		}

		if len(foundPositions) > 0 {
			offences = append(offences, &Offence{
				IdentDeprecation: d,
				Positions:        foundPositions,
			})
		}
	}

	return offences, nil
}

func findIdentifierPositions(fset *token.FileSet, nodes []ast.Node, ident *ast.Ident) ([]*token.Position, error) {
	positions := make([]*token.Position, 0, 0)

	for _, node := range nodes {
		nodeName := fmt.Sprint(node)
		if nodeName == ident.String() {
			position := fset.Position(node.Pos())
			positions = append(positions, &position)
		}
	}

	return positions, nil
}

func filesWhichImport(providerImportDetails *ProviderImportDetails, importPath string) (files []string, e error) {
	files = []string{}
	for _, p := range providerImportDetails.Packages {
		if util.StringSliceContains(p.Imports, importPath) {
			files = append(files, prependDirToFilePaths(p.GoFiles, p.Dir)...)
		}
		if util.StringSliceContains(p.TestImports, importPath) {
			files = append(files, prependDirToFilePaths(p.TestGoFiles, p.Dir)...)
		}
	}

	return files, nil
}

func prependDirToFilePaths(filePaths []string, dir string) []string {
	newFilePaths := []string{}
	for _, f := range filePaths {
		newFilePaths = append(newFilePaths, path.Join(dir, f))
	}
	return newFilePaths
}
