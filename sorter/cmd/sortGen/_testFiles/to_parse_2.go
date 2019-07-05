package main

//go:generate gobin -m -run myitcv.io/sorter/cmd/sortGen

import mysorter "myitcv.io/sorter"

// MATCH
func OrderByAge(persons []person, i, j int) mysorter.Ordered {
	return persons[i].age < persons[j].age
}
