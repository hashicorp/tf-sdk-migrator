# Reports

## Eligibility of providers

`tf-sdk-migrator --csv check` was run against all Terraform providers found in the `terraform-providers` GitHub org.

### Analysis

TBD once the train wifi lets me download csvkit.

### Data

Please see [./eligibility.csv](./eligibility.csv) for full data.

This file was generated with:

```sh
echo -n "provider," > eligibility.csv
go run . check --csv github.com/terraform-providers/terraform-provider-aws | head -n 1 >> eligibility.csv
for f in $GOPATH/src/github.com/terraform-providers/*; do echo -n $(basename $f), >> eligibility.csv; go run . check --csv github.com/terraform-providers/$(basename $f) | tail -n 1 >> eligibility.csv; done
```
