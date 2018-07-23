package protobuf

import "fmt"

// ImportPaths is a convenience type for use with the flag package to enable multiple
// import paths to be provided via multiple occurences of a flag
type ImportPaths []string

func (i *ImportPaths) Set(value string) error {
	*i = append(*i, value)
	return nil
}

func (i *ImportPaths) String() string {
	return fmt.Sprint(*i)
}
