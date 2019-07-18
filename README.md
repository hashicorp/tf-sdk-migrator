# tf-sdk-migrator

##

`REMOVED_PACKAGES` was created with this one-liner:

```sh
comm -13 <(cd terraform-plugin-sdk; go list ./... | sed -E 's/github.com\/hashicorp\/terraform-plugin-sdk\/sdk\/(internal\/)?//' | sort) <(cd ../terraform; go list ./... | sed 's/github.com\/hashicorp\/terraform\///' | sort) | xargs -I "%" echo "github.com/hashicorp/terraform/""%"
```

Working directory is `$GOPATH/src/github.com/hashicorp`, with `terraform` and `terraform-plugin-sdk` repos present.
