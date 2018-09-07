## TODO

* Documentation
* Support for embedded structs. At the moment this doesn't work because the embedding gets "converted" to a unexported field. So instead of relying on the methods sets being promoted, we will need to proxy them via the unexported field (which should be do-able)
* Examples:
  * Example of having immutable templates in a `*_test.go` file of package `a` and then using those types within xtest `a_test` package tests
