package consume2_test

import (
	"fmt"

	"github.com/keep94/consume2"
)

type Person struct {
	Name string
	Age  int
}

func FirstNamesOver40(people []Person, n int) (result []string) {
	over40 := consume2.PFilter(func(p Person) bool { return p.Age >= 40 })
	namesOver40 := consume2.Join(
		over40, consume2.PMap(func(p Person) string { return p.Name }))
	firstNamesOver40 := consume2.Join(
		namesOver40, consume2.PSlice[string](0, n))
	consume2.FromSlice(people, firstNamesOver40.AppendTo(&result))
	return
}

func Example_pipeline() {
	people := []Person{
		{Name: "Alice", Age: 43},
		{Name: "Bob", Age: 35},
		{Name: "Charlie", Age: 62},
		{Name: "David", Age: 40},
		{Name: "Ellen", Age: 41},
	}
	fmt.Println(FirstNamesOver40(people, 3))
	// Output:
	// [Alice Charlie David]
}
