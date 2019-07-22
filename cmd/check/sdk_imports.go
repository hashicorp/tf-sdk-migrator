package check

import (
	"os"
	"path/filepath"
	"strings"

	"github.com/kmoe/tf-sdk-migrator/util"
)

const REMOVED_PACKAGES = `github.com/hashicorp/terraform/backend
github.com/hashicorp/terraform/backend/atlas
github.com/hashicorp/terraform/backend/init
github.com/hashicorp/terraform/backend/local
github.com/hashicorp/terraform/backend/remote
github.com/hashicorp/terraform/backend/remote-state/artifactory
github.com/hashicorp/terraform/backend/remote-state/azure
github.com/hashicorp/terraform/backend/remote-state/consul
github.com/hashicorp/terraform/backend/remote-state/etcdv2
github.com/hashicorp/terraform/backend/remote-state/etcdv3
github.com/hashicorp/terraform/backend/remote-state/gcs
github.com/hashicorp/terraform/backend/remote-state/http
github.com/hashicorp/terraform/backend/remote-state/inmem
github.com/hashicorp/terraform/backend/remote-state/manta
github.com/hashicorp/terraform/backend/remote-state/oss
github.com/hashicorp/terraform/backend/remote-state/pg
github.com/hashicorp/terraform/backend/remote-state/s3
github.com/hashicorp/terraform/backend/remote-state/swift
github.com/hashicorp/terraform/builtin/bins/provider-test
github.com/hashicorp/terraform/builtin/bins/provisioner-chef
github.com/hashicorp/terraform/builtin/bins/provisioner-file
github.com/hashicorp/terraform/builtin/bins/provisioner-habitat
github.com/hashicorp/terraform/builtin/bins/provisioner-local-exec
github.com/hashicorp/terraform/builtin/bins/provisioner-puppet
github.com/hashicorp/terraform/builtin/bins/provisioner-remote-exec
github.com/hashicorp/terraform/builtin/bins/provisioner-salt-masterless
github.com/hashicorp/terraform/builtin/providers/terraform
github.com/hashicorp/terraform/builtin/providers/test
github.com/hashicorp/terraform/builtin/provisioners/chef
github.com/hashicorp/terraform/builtin/provisioners/file
github.com/hashicorp/terraform/builtin/provisioners/habitat
github.com/hashicorp/terraform/builtin/provisioners/local-exec
github.com/hashicorp/terraform/builtin/provisioners/puppet
github.com/hashicorp/terraform/builtin/provisioners/puppet/bolt
github.com/hashicorp/terraform/builtin/provisioners/remote-exec
github.com/hashicorp/terraform/builtin/provisioners/salt-masterless
github.com/hashicorp/terraform/command
github.com/hashicorp/terraform/command/clistate
github.com/hashicorp/terraform/command/e2etest
github.com/hashicorp/terraform/command/jsonconfig
github.com/hashicorp/terraform/command/jsonplan
github.com/hashicorp/terraform/command/jsonprovider
github.com/hashicorp/terraform/command/jsonstate
github.com/hashicorp/terraform/communicator
github.com/hashicorp/terraform/communicator/remote
github.com/hashicorp/terraform/communicator/shared
github.com/hashicorp/terraform/communicator/ssh
github.com/hashicorp/terraform/communicator/winrm
github.com/hashicorp/terraform/configs/configupgrade
github.com/hashicorp/terraform/digraph
github.com/hashicorp/terraform/e2e
github.com/hashicorp/terraform/flatmap
github.com/hashicorp/terraform/helper/diff
github.com/hashicorp/terraform/helper/shadow
github.com/hashicorp/terraform/helper/signalwrapper
github.com/hashicorp/terraform/helper/slowmessage
github.com/hashicorp/terraform/helper/variables
github.com/hashicorp/terraform/helper/wrappedreadline
github.com/hashicorp/terraform/helper/wrappedstreams
github.com/hashicorp/terraform/internal/earlyconfig
github.com/hashicorp/terraform/internal/initwd
github.com/hashicorp/terraform/internal/modsdir
github.com/hashicorp/terraform/internal/tfplugin5
github.com/hashicorp/terraform/repl
github.com/hashicorp/terraform/scripts
github.com/hashicorp/terraform/state
github.com/hashicorp/terraform/state/remote
github.com/hashicorp/terraform/states/statemgr
github.com/hashicorp/terraform/tools/loggraphdiff
github.com/hashicorp/terraform/tools/terraform-bundle
github.com/hashicorp/terraform/tools/terraform-bundle/e2etest`

func CheckSDKPackageImports(providerPath string) (removedPackagesInUse []string, doesNotUseRemovedPackages bool, e error) {
	removedPackages := strings.Split(REMOVED_PACKAGES, "\n")
	removedPackagesInUse = []string{}

	filepath.Walk(providerPath, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if info.IsDir() && info.Name() == "vendor" {
			return filepath.SkipDir
		}
		if !info.IsDir() && strings.HasSuffix(info.Name(), ".go") {
			removedPackagesInUse = append(removedPackagesInUse, util.FindImportedPackages(path, removedPackages)...)
		}
		return nil
	})

	return removedPackagesInUse, len(removedPackagesInUse) == 0, nil
}
