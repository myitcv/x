package main

type block string

func (b *block) String() string {
	if b == nil {
		return "nil"
	}

	return string(*b)
}
