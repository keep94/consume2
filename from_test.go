package consume2_test

import (
	"testing"

	"github.com/keep94/consume2"
	"github.com/stretchr/testify/assert"
)

func TestFromSlice(t *testing.T) {
	values := []int{3, 8, 9, 10, 15, 16, 23, 27}
	var evens []int
	consumer := consume2.Filter(
		consume2.AppendTo(&evens),
		func(x int) bool { return x%2 == 0 },
	)
	consume2.FromSlice(values, consumer)
	assert.Equal(t, []int{8, 10, 16}, evens)
}

func TestFromSlicePtr(t *testing.T) {
	people := []*person{
		{Name: "Alice", Age: 35},
		{Name: "Bobby", Age: 43},
		nil,
		{Name: "Sarah", Age: 46},
	}
	var over40 []*person
	consumer := consume2.Filter(
		consume2.AppendPtrsTo(&over40),
		func(p person) bool { return p.Age >= 40 },
	)
	consume2.FromPtrSlice(people, consumer)
	assert.Equal(
		t,
		[]*person{{Name: "Bobby", Age: 43}, {Name: "Sarah", Age: 46}},
		over40,
	)
}

func TestFromIntGenerator(t *testing.T) {
	var squares []int
	consume2.FromIntGenerator(
		squaresLessThan36(), consume2.AppendTo(&squares))
	assert.Equal(t, []int{1, 4, 9, 16, 25}, squares)
}

func TestFromGenerator(t *testing.T) {
	var cubes []int
	consume2.FromGenerator(
		cubesLessThan216(), consume2.AppendTo(&cubes))
	assert.Equal(t, []int{1, 8, 27, 64, 125}, cubes)
}

func squaresLessThan36() func() int {
	index := 1
	return func() int {
		if index == 6 {
			return -1
		}
		result := index * index
		index++
		return result
	}
}

func cubesLessThan216() func() (int, bool) {
	index := 1
	return func() (int, bool) {
		if index == 6 {
			return 0, false
		}
		result := index * index * index
		index++
		return result, true
	}
}
