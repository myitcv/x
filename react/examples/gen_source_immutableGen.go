// Code generated by immutableGen. DO NOT EDIT.

package examples

//immutableVet:skipFile

import (
	"myitcv.io/immutable"
)

//
// exampleSource is an immutable type and has the following template:
//
// 	map[exampleKey]*source
//
type exampleSource struct {
	theMap  map[exampleKey]*source
	mutable bool
	__tmpl  *_Imm_exampleSource
}

var _ immutable.Immutable = new(exampleSource)
var _ = new(exampleSource).__tmpl

func newExampleSource(inits ...func(m *exampleSource)) *exampleSource {
	res := newExampleSourceCap(0)
	if len(inits) == 0 {
		return res
	}

	return res.WithMutable(func(m *exampleSource) {
		for _, i := range inits {
			i(m)
		}
	})
}

func newExampleSourceCap(l int) *exampleSource {
	return &exampleSource{
		theMap: make(map[exampleKey]*source, l),
	}
}

func (m *exampleSource) Mutable() bool {
	return m.mutable
}

func (m *exampleSource) Len() int {
	if m == nil {
		return 0
	}

	return len(m.theMap)
}

func (m *exampleSource) Get(k exampleKey) (*source, bool) {
	v, ok := m.theMap[k]
	return v, ok
}

func (m *exampleSource) AsMutable() *exampleSource {
	if m == nil {
		return nil
	}

	if m.Mutable() {
		return m
	}

	res := m.dup()
	res.mutable = true

	return res
}

func (m *exampleSource) dup() *exampleSource {
	resMap := make(map[exampleKey]*source, len(m.theMap))

	for k := range m.theMap {
		resMap[k] = m.theMap[k]
	}

	res := &exampleSource{
		theMap: resMap,
	}

	return res
}

func (m *exampleSource) AsImmutable(v *exampleSource) *exampleSource {
	if m == nil {
		return nil
	}

	if v == m {
		return m
	}

	m.mutable = false
	return m
}

func (m *exampleSource) Range() map[exampleKey]*source {
	if m == nil {
		return nil
	}

	return m.theMap
}

func (mr *exampleSource) WithMutable(f func(e *exampleSource)) *exampleSource {
	res := mr.AsMutable()
	f(res)
	res = res.AsImmutable(mr)

	return res
}

func (mr *exampleSource) WithImmutable(f func(e *exampleSource)) *exampleSource {
	prev := mr.mutable
	mr.mutable = false
	f(mr)
	mr.mutable = prev

	return mr
}

func (m *exampleSource) Set(k exampleKey, v *source) *exampleSource {
	if m.mutable {
		m.theMap[k] = v
		return m
	}

	res := m.dup()
	res.theMap[k] = v

	return res
}

func (m *exampleSource) Del(k exampleKey) *exampleSource {
	if _, ok := m.theMap[k]; !ok {
		return m
	}

	if m.mutable {
		delete(m.theMap, k)
		return m
	}

	res := m.dup()
	delete(res.theMap, k)

	return res
}
func (s *exampleSource) IsDeeplyNonMutable(seen map[interface{}]bool) bool {
	if s == nil {
		return true
	}

	if s.Mutable() {
		return false
	}
	if s.Len() == 0 {
		return true
	}

	if seen == nil {
		return s.IsDeeplyNonMutable(make(map[interface{}]bool))
	}

	if seen[s] {
		return true
	}

	seen[s] = true

	for _, v := range s.theMap {
		if v != nil && !v.IsDeeplyNonMutable(seen) {
			return false
		}
	}
	return true
}

//
// source is an immutable type and has the following template:
//
// 	struct {
// 		file	string
// 		src	string
// 	}
//
type source struct {
	field_file string
	field_src  string

	mutable bool
	__tmpl  *_Imm_source
}

var _ immutable.Immutable = new(source)
var _ = new(source).__tmpl

func (s *source) AsMutable() *source {
	if s.Mutable() {
		return s
	}

	res := *s
	res.mutable = true
	return &res
}

func (s *source) AsImmutable(v *source) *source {
	if s == nil {
		return nil
	}

	if s == v {
		return s
	}

	s.mutable = false
	return s
}

func (s *source) Mutable() bool {
	return s.mutable
}

func (s *source) WithMutable(f func(si *source)) *source {
	res := s.AsMutable()
	f(res)
	res = res.AsImmutable(s)

	return res
}

func (s *source) WithImmutable(f func(si *source)) *source {
	prev := s.mutable
	s.mutable = false
	f(s)
	s.mutable = prev

	return s
}

func (s *source) IsDeeplyNonMutable(seen map[interface{}]bool) bool {
	if s == nil {
		return true
	}

	if s.Mutable() {
		return false
	}

	if seen == nil {
		return s.IsDeeplyNonMutable(make(map[interface{}]bool))
	}

	if seen[s] {
		return true
	}

	seen[s] = true
	return true
}
func (s *source) file() string {
	return s.field_file
}

// setFile is the setter for File()
func (s *source) setFile(n string) *source {
	if s.mutable {
		s.field_file = n
		return s
	}

	res := *s
	res.field_file = n
	return &res
}
func (s *source) src() string {
	return s.field_src
}

// setSrc is the setter for Src()
func (s *source) setSrc(n string) *source {
	if s.mutable {
		s.field_src = n
		return s
	}

	res := *s
	res.field_src = n
	return &res
}
