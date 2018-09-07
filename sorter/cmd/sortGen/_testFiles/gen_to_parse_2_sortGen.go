// Code generated by sortGen. DO NOT EDIT.

package main

import "sort"
import "myitcv.io/sorter"

func SortByAge(vs []person) {
	sort.Sort(&sorter.Wrapper{
		LenFunc: func() int {
			return len(vs)
		},
		LessFunc: func(i, j int) bool {
			return bool(OrderByAge(vs, i, j))
		},
		SwapFunc: func(i, j int) {
			vs[i], vs[j] = vs[j], vs[i]
		},
	})
}
func StableSortByAge(vs []person) {
	sort.Stable(&sorter.Wrapper{
		LenFunc: func() int {
			return len(vs)
		},
		LessFunc: func(i, j int) bool {
			return bool(OrderByAge(vs, i, j))
		},
		SwapFunc: func(i, j int) {
			vs[i], vs[j] = vs[j], vs[i]
		},
	})
}