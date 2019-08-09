package check

import (
	"fmt"
	"io/ioutil"

	version "github.com/hashicorp/go-version"
	"github.com/radeksimko/mod/modfile"
	"github.com/radeksimko/mod/module"
)

func ReadSDKVersionFromGoModFile(providerPath string) (*version.Version, error) {
	path := providerPath + "/go.mod"
	content, err := ioutil.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("could not read go.mod file for provider %s", providerPath)
	}

	pf, err := modfile.Parse(path, content, nil)
	if err != nil {
		return nil, err
	}

	mv, err := findMatchingRequireStmt(pf.Require, terraformDependencyPath)
	if err != nil {
		return nil, err
	}

	return version.NewVersion(mv.Version)
}

func findMatchingRequireStmt(requires []*modfile.Require, path string) (module.Version, error) {
	for _, requireStmt := range requires {
		mod := requireStmt.Mod
		if mod.Path == path {
			return mod, nil
		}
	}

	return module.Version{}, fmt.Errorf("require statement with path %q not found", path)
}
