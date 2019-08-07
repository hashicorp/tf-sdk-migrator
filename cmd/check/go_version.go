package check

import (
	"errors"
	"io/ioutil"
	"strings"

	version "github.com/hashicorp/go-version"
	"github.com/hashicorp/tf-sdk-migrator/util"
	"github.com/radeksimko/mod/modfile"
)

// .go-version file contains the Go version as a string, followed by \n
func ReadGoVersionFromGoVersionFile(providerPath string) (*version.Version, error) {
	content, err := ioutil.ReadFile(providerPath + "/.go-version")
	if err != nil {
		return nil, err
	}

	lines := strings.Split(string(content), "\n")

	v := strings.TrimSpace(lines[0])

	return version.NewVersion(v)
}

// go.mod file contains one directive per line
// the "go" directive sets the expected language version
// see https://tip.golang.org/cmd/go/#hdr-The_go_mod_file
func ReadGoVersionFromGoModFile(providerPath string) (*version.Version, error) {
	path := providerPath + "/go.mod"
	content, err := ioutil.ReadFile(path)
	if err != nil {
		return nil, err
	}

	pf, err := modfile.Parse(path, content, nil)
	if err != nil {
		return nil, err
	}

	if pf.Go == nil {
		return nil, errors.New("go statement not found")
	}

	return version.NewVersion(pf.Go.Version)
}

func ReadGoVersionFromTravisConfig(providerPath string) (*version.Version, error) {
	_, content, err := util.ReadOneOf(providerPath, "/.travis.yml", "/.travis.yaml")
	if err != nil {
		return nil, err
	}

	lines := strings.Split(string(content), "\n")

	if util.SearchLines(lines, "language: go", 0) == -1 {
		return nil, errors.New("no 'language: go' in travis config")
	}

	goLine := util.SearchLines(lines, "go:", 0)
	if goLine == -1 {
		return nil, errors.New("no 'go:' in travis config")
	}

	v := strings.TrimSpace(lines[goLine+1])
	v = strings.TrimLeft(v, ` -"`)
	v = strings.TrimRight(v, ` "x.`)

	return version.NewVersion(v)
}
