package check

import (
	"fmt"
	"go/ast"
	"go/token"

	"github.com/radeksimko/go-refs/parser"
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
}

func CheckSDKPackageRefs() ([]*Offence, error) {
	offences := make([]*Offence, 0, 0)

	for _, d := range deprecations {
		files, err := filesWhichImport(d.ImportPath)
		if err != nil {
			return nil, err
		}

		foundPositions := make([]*token.Position, 0, 0)

		for _, filePath := range files {
			f, err := parser.ParseFile(filePath)
			if err != nil {
				return nil, err
			}

			identifiers, err := parser.FindPackageReferences(f, d.ImportPath)
			if err != nil {
				return nil, err
			}

			positions, err := findIdentifierPositions(identifiers, d.Identifier)
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

func findIdentifierPositions(nodes []ast.Node, ident *ast.Ident) ([]*token.Position, error) {
	fset := token.NewFileSet()
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

func filesWhichImport(importPath string) ([]string, error) {
	return []string{}, nil
}
