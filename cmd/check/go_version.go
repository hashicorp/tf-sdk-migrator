package check

import (
	"errors"
	"io/ioutil"
	"strings"

	version "github.com/hashicorp/go-version"
	"github.com/kmoe/tf-sdk-migrator/util"
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
	content, err := ioutil.ReadFile(providerPath + "/go.mod")
	if err != nil {
		return nil, err
	}

	lines := strings.Split(string(content), "\n")

	goLine := util.SearchLinesPrefix(lines, "go ", 0)
	if goLine == -1 {
		return nil, errors.New("no 'go' directive in go.mod")
	}

	v := strings.TrimLeft(lines[goLine], "go ")
	v = strings.TrimSpace(v)

	return version.NewVersion(v)
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
