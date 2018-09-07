## `immutableGen`

`immutableGen` is an [untyped Go generator](https://github.com/myitcv/x/blob/master/gogenerate/_doc/README.md) that generates immutable struct, slice and map types from template type specs that have an `_Imm_*` prefix. All such generated types "implement" a common immutable "interface" as well as providing functions and methods specific to the type (struct, map or slice).

_Note we quoted use of the terms "implement" and "interface"; as you will see below, the "interface" described is a Go interface presented as if the language, and interfaces in particular, supported [generics](https://en.wikipedia.org/wiki/Generic_programming). The authors are quite comfortable with the lack of generics in the language, we instead use this technique as a succinct way of documenting the behaviour of `immutableGen`_

## The common immutable "interface"

Immutable struct, slice and map types all "implement" the following "interface":

```go
type Immutable /*<T>*/ interface {

   // Mutable indicates whether the receiver is mutable or not
   //
   Mutable() bool

   // IsDeeplyNonMutable indicates whether the receiver and all immutable type
   // values it transitively references are not Mutable()
   //
   IsDeeplyNonMutable(seen map[interface{}]bool) bool

   // AsMutable returns a mutable copy of the receiver in case the receiver is
   // immutable, or the receiver in case it is already mutable.
   //
   AsMutable() *T

   // AsImmutable is used to mark the receiver as immutable. The receiver is
   // marked immutable based on whether prev is immutable or not; in case prev
   // is immutable (or nil), then the receiver is marked immutable. Otherwise
   // the receiver is left in its current state.  In either case the receiver
   // is returned.
   //
   AsImmutable(prev *T) *T

   // WithMutable applies f to the mutable result of z :=
   // receiver.AsMutable(), and then returns z.AsImmutable(receiver). It is
   // effectively a convenience convenience wrapper around AsMutable() and
   // AsImmutable().
   //
   WithMutable(f func(t *T)) *T

   // WithImmutable applies f to the receiver marked as immutable, and returns
   // the receiver in its original state (mutable or immutable).
   //
   WithImmutable(f func(t *T)) *T
}
```

## Immutable structs

Taking the following as a simple example:

```go
type _Imm_Banana struct {
   Name string
   Age  int
}
```

For each field in this template, with name `N` and type `X`, the generated immutable struct "implements" the following "interface":

```go
type ImmutableStructField /*<T, X>*/ interface {

   // N returns the field value for N.
   N() X

   // SetN set the value x in the field N on a mutable copy of the receiver
   // z := AsMutable(), and then returns z.AsImmutable(receiver)
   //
   SetN(x X) *T
}
```

## Immutable slices

Considering the template:

```go
type _Imm_T []V
```

then the resulting type `T` "implements" the immutable slice "interface":

```go
// NewT returns an immutable slice, with Mutable() == false, containing the
// provided elements
//
func NewT(s ...V) *T {}

// NewTLen returns an immutable slice, with Mutable() == false, of length
//
func NewTLen(l int) *T {}

type ImmutableSlice /*<T, V>*/ interface {

   // Len returns the length of the immutable slice.
   //
   Len() int

   // Get returns the value at index i from the immutable slice.
   //
   Get(i int) V

   // Set sets the value v at index i in the mutable result z :=
   // receiver.AsMutable(), and then returns z.AsImmutable(receiver).
   //
   Set(i int, v V) *T

   // Range is used to range over the values in an immutable slice.
   //
   Range() []V

   // Append appends the provided values to the mutable result z :=
   // receiver.AsMutable(), and then returns z.AsImmutable(receiver)
   //
   Append(v ...V) *T
}
```

## Immutable maps

Considering the template:

```go
type _Imm_T map[K]V
```

then the resulting type `T` "implements" the immutable map "interface":


```go
// NewT returns an immutable map with capacity 0 if there are no arguments.
// Where len(inits) > 0, NewT returns the result of applying each of the
// init functions to the value v := NewT().AsMutable(), before returning the
// value v.AsImmutable(nil)
//
func NewT(inits ...func(m *T)) *T {}

// NewTCap returns an immutable map with capacity l.
//
func NewTCap(l int) *T {}

type ImmutableMap /*<T, K, V>*/ interface {

   // Len returns the length of the immutable map.
   Len() int

   // Get returns the immtable map value with key k and true if the immutable
   // map contains an entry with key k, or the zero value of type V and false
   // otherwise
   //
   Get(k *K) (V, bool)

   // Set (need to finish...)
   //
   Set(k *K, v *V) *T

   // Del (need to finish...)
   //
   Del(k *K) *T

   // Range (need to finish...)
   //
   Range() map[K]V
}
```
