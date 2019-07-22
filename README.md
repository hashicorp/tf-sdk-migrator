# tf-sdk-migrator

The Terraform provider plugin SDK, previously part of the [github.com/hashicorp/terraform](https://github.com/hashicorp/terraform) "Core" Go module, has been moved to a new Go module, [github.com/hashicorp/terraform-plugin-sdk](https://github.com/hashicorp/terraform-plugin-sdk). Terraform providers will in future import `hashcorp/terraform-plugin-sdk`.

`tf-sdk-migrator` is a CLI tool which will migrate a Terraform provider to the new SDK module by rewriting import paths. `tf-sdk-migrator check` checks the eligibility of the Provider for migration.

Eligibility data for current providers can be found in [./reports](./reports).

## Usage

### Build

```sh
go build
./tf-sdk-migrator
```

### Check eligibility for migration: `tf-sdk-migrator check`

Checks whether a Terraform provider is ready to migrate to the newly extracted Terraform SDK package. 

```sh
tf-sdk-migrator check [--help] PATH
```

Outputs a report containing:
 - Go version used in provider
 - Whether the provider uses Go modules
 - Version of `hashicorp/terraform` used
 - Whether the provider uses any `hashicorp/terraform` packages that are not in `hashicorp/terraform-plugin-sdk`
 
The `--csv` flag will output values in CSV format.

Exits 0 if the provider meets all the migration requirements, 1 otherwise.

### Migrate to SDK module

Docs to be added.

---

## Development

### Listing packages in Core but not SDK

`REMOVED_PACKAGES` in `cmd/check/sdk_imports.go` was created with this one-liner:

```sh
comm -13 <(cd terraform-plugin-sdk; go list ./... | sed -E 's/github.com\/hashicorp\/terraform-plugin-sdk\/sdk\/(internal\/)?//' | sort) <(cd ../terraform; go list ./... | sed 's/github.com\/hashicorp\/terraform\///' | sort) | xargs -I "%" echo "github.com/hashicorp/terraform/""%"
```

Working directory is `$GOPATH/src/github.com/hashicorp`, with `terraform` and `terraform-plugin-sdk` repos present.
