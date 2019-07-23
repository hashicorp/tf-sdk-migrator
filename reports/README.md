# Reports

## Eligibility of providers

`tf-sdk-migrator --csv check` was run against the `master` branch of all Terraform providers found in the `terraform-providers` GitHub org, excepting `terraform-provider-cidr` (which is currently empty) and `terraform-provider-scaffolding` (which is not a provider).

### Analysis

This analysis was performed using [csvkit](https://csvkit.readthedocs.io).

#### Go version

Of the 110 providers analysed:
 - 40 use Go version 1.12.x
 - 69 use Go version 1.11.x
 - 1 (`terraform-provider-ciscoasa`) has no Go version listed
 
#### Go modules

Of the 110 providers analysed, all use Go modules.
 
#### SDK version

Of the 110 providers analysed:
 - 69 use version `0.12.x`
 - 16 use a prerelease version of `0.12`
 - 25 use earlier versions

#### Use of removed SDK packages

Providers currently using removed SDK packages are:
 - `terraform-provider-aws`
 - `terraform-provider-nutanix`
 - `terraform-provider-runscope`

All of these use only `github.com/hashicorp/terraform/flatmap`.

 - `terraform-provider-terraform`

This provider uses `github.com/hashicorp/terraform/backend` and `github.com/hashicorp/terraform/backend/init`.

#### Overall eligibility

Of the 110 providers analysed, 23 satisfy all requirements for upgrading to the new SDK, and 87 do not.


### Data

Please see [./eligibility.csv](./eligibility.csv) for full data.

This file was generated with [./generate_eligibility_csv.sh](./generate_eligibility_csv.sh).
