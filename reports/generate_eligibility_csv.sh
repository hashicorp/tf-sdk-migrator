#!/bin/bash

echo -n "provider," > eligibility.csv

../tf-sdk-migrator check --csv github.com/terraform-providers/terraform-provider-aws | head -n 1 >> eligibility.csv

for f in $GOPATH/src/github.com/terraform-providers/*; do echo -n $(basename $f), >> eligibility.csv; ../tf-sdk-migrator check --csv github.com/terraform-providers/$(basename $f) | tail -n 1 >> eligibility.csv; done
