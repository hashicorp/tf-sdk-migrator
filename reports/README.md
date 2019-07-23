# Reports

## Eligibility of providers

`tf-sdk-migrator --csv check` was run against all Terraform providers found in the `terraform-providers` GitHub org.

### Analysis

#### Use of removed SDK packages

Providers currently using removed SDK packages are:
 - `terraform-provider-aws`
 - `terraform-provider-nutanix`
 - `terraform-provider-runscope`

All of these use only `github.com/hashicorp/terraform/flatmap`.

 - `terraform-provider-terraform`

This provider uses `github.com/hashicorp/terraform/backend` and `github.com/hashicorp/terraform/backend/init`.

### Data

Please see [./eligibility.csv](./eligibility.csv) for full data.

This file was generated with:

```sh
echo -n "provider," > eligibility.csv
go run . check --csv github.com/terraform-providers/terraform-provider-aws | head -n 1 >> eligibility.csv
for f in $GOPATH/src/github.com/terraform-providers/*; do echo -n $(basename $f), >> eligibility.csv; go run . check --csv github.com/terraform-providers/$(basename $f) | tail -n 1 >> eligibility.csv; done
```
