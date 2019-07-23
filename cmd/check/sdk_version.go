package check

import (
	"errors"
	"io/ioutil"
	"strings"

	"github.com/hashicorp/go-multierror"
	version "github.com/hashicorp/go-version"
	"github.com/hashicorp/tf-sdk-migrator/util"
)

func ReadSDKVersionFromGoModFile(providerPath string) (*version.Version, error) {
	content, err := ioutil.ReadFile(providerPath + "/go.mod")
	if err != nil {
		return nil, multierror.Append(err, errors.New("could not read go.mod file for provider "+providerPath))
	}

	lines := strings.Split(string(content), "\n")

	terraformPackageLine := util.SearchLines(lines, terraformDependencyPath+" ", 0)
	if terraformPackageLine == -1 {
		return nil, errors.New("could not find github/hashicorp/terraform dependency for provider " + providerPath)
	}

	v := strings.TrimSpace(lines[terraformPackageLine])
	v = strings.TrimLeft(v, "require ")
	v = strings.TrimLeft(v, terraformDependencyPath+" ")

	return version.NewVersion(v)
}
