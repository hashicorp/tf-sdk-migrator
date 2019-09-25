# tf-sdk-migrator

The Terraform provider plugin SDK, previously part of the [github.com/hashicorp/terraform](https://github.com/hashicorp/terraform) "Core" Go module, has been moved to a new Go module, [github.com/hashicorp/terraform-plugin-sdk](https://github.com/hashicorp/terraform-plugin-sdk). Terraform providers should now import `hashcorp/terraform-plugin-sdk`.

`tf-sdk-migrator` is a CLI tool which will migrate a Terraform provider to the new SDK module by rewriting import paths. `tf-sdk-migrator check` checks the eligibility of the Provider for migration.

## Usage

```sh
go install github.com/hashicorp/tf-sdk-migrator
$GOBIN/tf-sdk-migrator
```

### Build

```sh
go build
./tf-sdk-migrator
```

### Check eligibility for migration: `tf-sdk-migrator check`

Checks whether a Terraform provider is ready to migrate to the newly extracted Terraform SDK package. 

```sh
tf-sdk-migrator check [--help] [--csv] PATH
```

Outputs a report containing:
 - Go version used in provider (soft requirement)
 - Whether the provider uses Go modules
 - Version of `hashicorp/terraform` used
 - Whether the provider uses any `hashicorp/terraform` packages that are not in `hashicorp/terraform-plugin-sdk`
 
The `--csv` flag will output values in CSV format.

Exits 0 if the provider meets all the hard requirements, 1 otherwise.

The Go version requirement is a "soft" requirement: it is strongly recommended to upgrade to Go version 1.12+ before migrating to the new SDK, but the migration can still be performed if this requirement is not met.

### Migrate to SDK module: `tf-sdk-migrator migrate`

Migrates the Terraform provider to the new extracted SDK (`github.com/hashicorp/terraform-plugin-sdk`), replacing references to the old SDK (`github.com/hashicorp/terraform`).

**Note: No backup is made before modifying files. Please make sure your VCS staging area is clean.**

```sh
tf-sdk-migrator migrate [--help] PATH
```

The eligibility check will be run first: migration will not proceed if this check fails.

The migration tool will then make the following changes:
 - `go.mod`: replace `github.com/hashicorp/terraform` dependency with `github.com/hashicorp/terraform-plugin-sdk`
 - rewrite import paths in all provider `.go` files (except in `vendor/`) accordingly
 - run `go mod tidy`

If you use vendored Go dependencies, you should run `go mod vendor` afterwards.

