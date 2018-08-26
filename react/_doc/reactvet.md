## `reactVet`

`reactVet` is a vet program used to check the correctness of `myitcv.io/react`-based packages.

Like the `go` tool, it takes a list of packages as arguments:

```bash
reactVet ./...
```

If no packages are provided, the current directory is assumed as the package to test.
