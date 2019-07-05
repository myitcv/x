package model

//go:generate gobin -m -run myitcv.io/immutable/cmd/immutableGen

type _Imm_Person struct {
	Name string
	Age  int
}

type _Imm_People []*Person

func NewPerson(name string, age int) *Person {
	return new(Person).WithMutable(func(p *Person) {
		p.SetName(name)
		p.SetAge(age)
	})
}
