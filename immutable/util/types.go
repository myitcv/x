package util

import (
	"go/types"
	"sync"

	"golang.org/x/tools/go/types/typeutil"
)

type ImmType interface {
	isImmType()
}

type (
	// ImmTypeBasic identifies Go types that are inherently immutable, e.g.
	// ints, strings
	ImmTypeBasic struct{}

	// ImmTypeStruct is used to indicate a type that is immutable by virtue of
	// being a pointer to a struct type that was itself generated from an _Imm_
	// struct template.
	ImmTypeStruct struct {
		Struct *types.Struct
	}

	// ImmTypeMap is used to indicate a type that is immutable by virtue of
	// being a pointer to a struct type that was itself generated from an _Imm_
	// map template.
	ImmTypeMap struct {
		Key  types.Type
		Elem types.Type
	}

	// ImmTypeMap is used to indicate a type that is immutable by virtue of
	// being a pointer to a struct type that was itself generated from an _Imm_
	// slice template.
	ImmTypeSlice struct {
		Elem types.Type
	}

	// ImmTypeImplsIntf is used to indicate a type that is not an ImmTypeStruct,
	// ImmTypeMap or ImmTypeSlice, but still satisfies the immutable "interface".
	// See the docs for myitcv.io/immutable.Immutable.
	ImmTypeImplsIntf struct{}

	// ImmTypeSimple is used to indiciate an interface type that extends the
	// myitcv.io/immutable.Immutable interface.
	ImmTypeSimple struct{}
)

func (i ImmTypeBasic) isImmType()     {}
func (i ImmTypeStruct) isImmType()    {}
func (i ImmTypeMap) isImmType()       {}
func (i ImmTypeSlice) isImmType()     {}
func (i ImmTypeImplsIntf) isImmType() {}
func (i ImmTypeSimple) isImmType()    {}

// TODO make a non-global; define a good API for creating a new checker
// after we have a couple of use cases that break with this approach.
var ic = &immCache{
	msCache: new(typeutil.MethodSetCache),
	res:     make(map[types.Type]ImmType),
}

type immCache struct {
	mu sync.Mutex

	// not entirely clear we even need this because we cache
	// the results of determining whether a pointer type is immutable
	// or not.
	msCache *typeutil.MethodSetCache

	// res is a cache of the non-pointer type to the result
	// because pointer type values are not comparable
	res map[types.Type]ImmType
}

func (i *immCache) lookup(tt types.Type) (v ImmType) {
	var cacheKey = tt

	// fast path for Go types that are inherently immutable
	switch tt := tt.Underlying().(type) {
	case *types.Basic:
		return ImmTypeBasic{}
	case *types.Pointer:
		cacheKey = tt.Elem()
	case *types.Interface:
		// ideally we would use types.Implements here... but we don't have a
		// reference to myitcv.io/immutable.Immutable. So we do it by hand for now.
	default:
		// see comment below
		return nil
	}

	// From this point onwards we have to implement the immutable "interface". And to my best
	// understanding at this point in time, that is only possible if the type is a pointer.
	// Hence anything else cannot be an immutable type.

	// We don't actually care whether we are pointing to a named type or not... because we use
	// underlying below.

	i.mu.Lock()
	defer i.mu.Unlock()

	v, ok := i.res[cacheKey]
	if ok {
		return v
	}

	defer func() {
		i.res[cacheKey] = v
	}()

	ms := i.msCache.MethodSet(tt)

	foundMutable := false
	foundAsMutable := false
	foundAsImmutable := false
	foundWithMutable := false
	foundWithImmutable := false
	foundIsDeeply := false

	pt, ptOk := tt.(*types.Pointer)

	isPtrToSelf := func(t types.Type) bool {
		if !ptOk {
			return false
		}

		ppt, ok := t.(*types.Pointer)
		if !ok {
			return false
		}

		return ppt.Elem() == pt.Elem()
	}

	for i := 0; i < ms.Len(); i++ {
		f := ms.At(i).Obj().(*types.Func)
		t := f.Type().(*types.Signature)

		switch mn := f.Name(); mn {
		case "Mutable":
			if t.Params().Len() != 0 {
				break
			}

			if t.Results().Len() != 1 {
				break
			}

			tres := t.Results().At(0)

			if b, ok := tres.Type().(*types.Basic); ok {
				foundMutable = b.Kind() == types.Bool
			}
		case "AsMutable":
			if t.Params().Len() != 0 {
				break
			}

			if t.Results().Len() != 1 {
				break
			}

			foundAsMutable = isPtrToSelf(t.Results().At(0).Type())

		case "AsImmutable":
			if t.Params().Len() != 1 {
				break
			}

			if !isPtrToSelf(t.Params().At(0).Type()) {
				break
			}

			if t.Results().Len() != 1 {
				break
			}

			foundAsImmutable = isPtrToSelf(t.Results().At(0).Type())

		case "WithMutable", "WithImmutable":
			if t.Params().Len() != 1 {
				break
			}

			st, ok := t.Params().At(0).Type().(*types.Signature)
			if !ok {
				break
			}

			if st.Params().Len() != 1 {
				break
			}

			if !isPtrToSelf(st.Params().At(0).Type()) {
				break
			}

			if st.Results().Len() != 0 {
				break
			}

			if t.Results().Len() != 1 {
				break
			}

			valid := isPtrToSelf(t.Results().At(0).Type())

			switch mn {
			case "WithMutable":
				foundWithMutable = valid
			case "WithImmutable":
				foundWithImmutable = valid
			}
		case "IsDeeplyNonMutable":
			if t.Params().Len() != 1 {
				break
			}

			mt, ok := t.Params().At(0).Type().(*types.Map)
			if !ok {
				break
			}

			if it, ok := mt.Key().(*types.Interface); !ok || !it.Empty() {
				break
			}

			if t.Results().Len() != 1 {
				break
			}

			foundIsDeeply = t.Results().At(0).Type() == types.Typ[types.Bool]
		}

	}

	isImm := foundMutable && foundAsMutable && foundAsImmutable &&
		foundWithMutable && foundWithImmutable && foundIsDeeply

	isImmSimple := foundMutable && foundIsDeeply

	if !isImm {
		if isImmSimple {
			v = ImmTypeSimple{}
		}
		return
	}

	v = ImmTypeImplsIntf{}

	// now we work out whether it's a struct, slice of map... else
	// it's unknown to this package

	st, ok := pt.Elem().Underlying().(*types.Struct)
	if !ok {
		return
	}

	hasTmpl := false

	// TODO this could probably be a bit more robust
	// but we use this fairly coarse mechanism to determine
	// whether the struct we have in hand is the result of
	// immutableGen generation by looking for well-known fields
	for i := 0; i < st.NumFields(); i++ {
		f := st.Field(i)

		switch f.Name() {
		case "__tmpl":
			hasTmpl = true
		case "theMap":
			m := f.Type().(*types.Map)
			v = ImmTypeMap{
				Key:  m.Key(),
				Elem: m.Elem(),
			}
		case "theSlice":
			s := f.Type().(*types.Slice)
			v = ImmTypeSlice{
				Elem: s.Elem(),
			}
		}
	}

	if v == (ImmTypeImplsIntf{}) && hasTmpl {
		v = ImmTypeStruct{
			Struct: st,
		}
	}

	return
}

// IsImmType determines whether the supplied type is an immutable type. In case
// a type is immutable, a value of type ImmTypeStruct, ImmTypeSlice or
// ImmTypeMap is returned. In case the type is immutable but neither of the
// aforementioned instances, ImmTypeUnknown is returned. If a type is not
// immutable then nil is returned
func IsImmType(t types.Type) ImmType {
	return ic.lookup(t)
}
