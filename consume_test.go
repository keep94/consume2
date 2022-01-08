package consume2_test

import (
	"strconv"
	"testing"

	"github.com/keep94/consume2"
	"github.com/stretchr/testify/assert"
)

type person struct {
	Name string
	Age  int
}

const (
	mark = iota
	stoney
	matt
	dillon
	beth
)

var people = []person{
	{Name: "Mark", Age: 50},
	{Name: "Stoney", Age: 49},
	{Name: "Matt", Age: 46},
	{Name: "Dillon", Age: 19},
	{Name: "Beth", Age: 54},
}

func TestNil(t *testing.T) {
	assert := assert.New(t)
	consumer := consume2.Nil[int]()
	assert.False(consumer.CanConsume())
	assert.Panics(func() { consumer.Consume(7) })
}

func TestMustCanConsume(t *testing.T) {
	assert := assert.New(t)
	nilConsumer := consume2.Nil[int]()
	assert.Panics(func() { consume2.MustCanConsume(nilConsumer) })
	var x []int
	consumer := consume2.AppendTo(&x)
	assert.NotPanics(func() { consume2.MustCanConsume(consumer) })
}

func TestConsumerFunc(t *testing.T) {
	assert := assert.New(t)
	var x int
	consumer := consume2.ConsumerFunc[int](func(value int) {
		x += value
	})
	consumer.Consume(4)
	consumer.Consume(5)
	assert.Equal(9, x)
	assert.True(consumer.CanConsume())
}

func TestPageBuilder_PageZero(t *testing.T) {
	assert := assert.New(t)
	pager := consume2.NewPageBuilder[int](0, 5)
	feedInts(t, pager)
	values, morePages := pager.Build()
	assert.Equal([]int{0, 1, 2, 3, 4}, values)
	assert.True(morePages)
}

func TestPageBuilder_PageThree(t *testing.T) {
	assert := assert.New(t)
	pager := consume2.NewPageBuilder[int](3, 5)
	feedInts(t, pager)
	values, morePages := pager.Build()
	assert.Equal([]int{15, 16, 17, 18, 19}, values)
	assert.True(morePages)
}

func TestPageBuilder_NoMorePages(t *testing.T) {
	assert := assert.New(t)
	pager := consume2.NewPageBuilder[int](2, 5)
	feedInts(t, consume2.Slice[int](pager, 0, 15))
	values, morePages := pager.Build()
	assert.Equal([]int{10, 11, 12, 13, 14}, values)
	assert.False(morePages)
}

func TestPageBuilder_PartialPage(t *testing.T) {
	assert := assert.New(t)
	pager := consume2.NewPageBuilder[int](2, 5)
	feedInts(t, consume2.Slice[int](pager, 0, 11))
	values, morePages := pager.Build()
	assert.Equal([]int{10}, values)
	assert.False(morePages)
}

func TestPageBuilder_EmptyPage(t *testing.T) {
	assert := assert.New(t)
	pager := consume2.NewPageBuilder[int](2, 5)
	feedInts(t, consume2.Slice[int](pager, 0, 10))
	values, morePages := pager.Build()
	assert.Equal([]int{}, values)
	assert.False(morePages)
}

func TestNewPageBuilderPanics(t *testing.T) {
	assert := assert.New(t)
	assert.Panics(func() { consume2.NewPageBuilder[int](0, -1) })
	assert.Panics(func() { consume2.NewPageBuilder[int](0, 0) })
	assert.Panics(func() { consume2.NewPageBuilder[int](-1, 5) })
}

func TestComposeEmpty(t *testing.T) {
	assert := assert.New(t)
	consumer := consume2.Compose[int]()
	assert.False(consumer.CanConsume())
	assert.Panics(func() { consumer.Consume(7) })
}

func TestComposeOne(t *testing.T) {
	assert := assert.New(t)
	var zeroTo5 []int
	consumer := consume2.Compose(consume2.AppendTo(&zeroTo5))
	feedInts(t, consume2.Slice(consumer, 0, 5))
	assert.Equal([]int{0, 1, 2, 3, 4}, zeroTo5)
}

func TestComposeUseIndividual(t *testing.T) {
	assert := assert.New(t)
	var strs []string
	var ints []int
	consumerOne := consume2.Map(
		consume2.Slice(consume2.AppendTo(&strs), 0, 1), strconv.Itoa)
	consumerThree := consume2.Slice(consume2.AppendTo(&ints), 0, 3)
	composite := consume2.Compose(
		consumerOne, consumerThree, consume2.Nil[int]())
	assert.True(composite.CanConsume())
	composite.Consume(1)
	assert.True(composite.CanConsume())
	composite.Consume(2)
	assert.True(composite.CanConsume())

	// Use up individual consumer
	consumerThree.Consume(3)

	// Now the composite consumer should return false
	assert.False(composite.CanConsume())

	assert.Equal([]string{"1"}, strs)
	assert.Equal([]int{1, 2, 3}, ints)
}

func TestComposeDefensiveCopy(t *testing.T) {
	assert := assert.New(t)
	var x, y []int
	consumers := []consume2.Consumer[int]{
		consume2.AppendTo(&x), consume2.AppendTo(&y)}
	composite := consume2.Compose(consumers...)

	// Mutating consumers shouldn't affect composite
	consumers[0] = nil

	feedInts(t, consume2.Slice(composite, 0, 1))
	assert.Equal([]int{0}, x)
	assert.Equal([]int{0}, y)
}

func TestSlice(t *testing.T) {
	assert := assert.New(t)
	var threeToSeven []int
	feedInts(t, consume2.Slice(consume2.AppendTo(&threeToSeven), 3, 7))
	assert.Equal([]int{3, 4, 5, 6}, threeToSeven)
}

func TestSliceNegative(t *testing.T) {
	assert := assert.New(t)
	var zeroToFive []int
	feedInts(t, consume2.Slice(consume2.AppendTo(&zeroToFive), -1, 5))
	assert.Equal([]int{0, 1, 2, 3, 4}, zeroToFive)
	var none []int
	feedInts(t, consume2.Slice(consume2.AppendTo(&none), 5, -1))
	feedInts(t, consume2.Slice(consume2.AppendTo(&none), -3, -1))
	feedInts(t, consume2.Slice(consume2.AppendTo(&none), -1, -3))
	feedInts(t, consume2.Slice(consume2.AppendTo(&none), -2, 0))
	feedInts(t, consume2.Slice(consume2.AppendTo(&none), 0, -2))
	assert.Empty(none)
}

func TestFilter(t *testing.T) {
	assert := assert.New(t)
	var sevensTo28 []int
	feedInts(t, consume2.Filter(
		consume2.Slice(consume2.AppendTo(&sevensTo28), 1, 4),
		func(value int) bool { return value%7 == 0 }))
	assert.Equal([]int{7, 14, 21}, sevensTo28)
}

func TestFilterp(t *testing.T) {
	assert := assert.New(t)
	var fiftiesTo300 []int
	feedInts(t, consume2.Filterp(
		consume2.Slice(consume2.AppendTo(&fiftiesTo300), 1, 6),
		func(ptr *int) bool {
			if (*ptr)%5 != 0 {
				return false
			}
			*ptr *= 10
			return true
		}))
	assert.Equal([]int{50, 100, 150, 200, 250}, fiftiesTo300)
}

func TestMap(t *testing.T) {
	assert := assert.New(t)
	var zeroTo5 []string
	feedInts(t, consume2.Map(
		consume2.Slice(consume2.AppendTo(&zeroTo5), 0, 5),
		strconv.Itoa))
	assert.Equal([]string{"0", "1", "2", "3", "4"}, zeroTo5)
}

func TestMaybeMap(t *testing.T) {
	assert := assert.New(t)
	var zeroTo10By2 []string
	feedInts(t, consume2.MaybeMap(
		consume2.Slice(consume2.AppendTo(&zeroTo10By2), 0, 5),
		func(value int) (str string, ok bool) {
			if value%2 == 1 {
				return
			}
			return strconv.Itoa(value), true
		}))
	assert.Equal([]string{"0", "2", "4", "6", "8"}, zeroTo10By2)
}

func TestTakeWhile(t *testing.T) {
	assert := assert.New(t)
	var zeroTo4 []int
	feedInts(t, consume2.TakeWhile(
		consume2.AppendTo(&zeroTo4),
		func(value int) bool { return value < 4 }))
	assert.Equal([]int{0, 1, 2, 3}, zeroTo4)
}

func TestTakeWhileInnerFinishes(t *testing.T) {
	assert := assert.New(t)
	var zeroTo4 []int
	feedInts(t, consume2.TakeWhile(
		consume2.Slice(consume2.AppendTo(&zeroTo4), 0, 4),
		func(value int) bool { return true }))
	assert.Equal([]int{0, 1, 2, 3}, zeroTo4)
}

func TestComposeFiltersNone(t *testing.T) {
	assert := assert.New(t)
	filter := consume2.ComposeFilters[int]()
	assert.True(filter(7))
}

func TestComposeFiltersOne(t *testing.T) {
	assert := assert.New(t)
	filter := consume2.ComposeFilters(
		func(value int) bool { return value%2 == 0 })
	assert.True(filter(8))
	assert.False(filter(7))
}

func TestComposeFiltersDefensiveCopy(t *testing.T) {
	assert := assert.New(t)
	filters := []func(int) bool{
		func(val int) bool { return val%2 == 0 },
		func(val int) bool { return val%3 == 0 },
	}
	composite := consume2.ComposeFilters(filters...)

	// mutating filters shouldn't affect composite
	filters[0] = nil

	assert.True(composite(6))
	assert.False(composite(5))
}

func TestComposeFiltersp(t *testing.T) {
	assert := assert.New(t)
	filter := consume2.ComposeFilters(
		func(ptr *int) bool { return (*ptr)%2 == 0 },
		func(ptr *int) bool { return (*ptr)%3 == 0 },
		func(ptr *int) bool {
			*ptr *= 10
			return (*ptr)%50 == 0
		},
	)
	var to1200By300 []int
	feedInts(t, consume2.TakeWhile(
		consume2.Filterp(consume2.AppendTo(&to1200By300), filter),
		func(value int) bool { return value < 120 }))
	assert.Equal([]int{0, 300, 600, 900}, to1200By300)
}

func TestComposeFilters(t *testing.T) {
	assert := assert.New(t)
	filter := consume2.ComposeFilters(
		func(value int) bool { return value%2 == 0 },
		func(value int) bool { return value%3 == 0 },
		func(value int) bool { return value%5 == 0 },
	)
	var to120By30 []int
	feedInts(t, consume2.TakeWhile(
		consume2.Filter(consume2.AppendTo(&to120By30), filter),
		func(value int) bool { return value < 120 }))
	assert.Equal([]int{0, 30, 60, 90}, to120By30)
}

func TestTrickyConsume(t *testing.T) {
	assert := assert.New(t)
	filter := func(ptr *person) bool {
		ptr.Age *= 2
		return true
	}
	var x, y []person
	consumer := consume2.Filterp(
		consume2.Compose(
			consume2.Filterp(consume2.AppendTo(&x), filter),
			consume2.AppendTo(&y),
		),
		filter,
	)
	consumer.Consume(people[beth])
	assert.Equal([]person{{Name: "Beth", Age: 216}}, x)
	assert.Equal([]person{{Name: "Beth", Age: 108}}, y)
}

func TestAppendPtrsTo(t *testing.T) {
	assert := assert.New(t)
	var result []*person
	consumer := consume2.AppendPtrsTo(&result)
	writePeopleInLoop(people[:], consume2.Slice(consumer, 0, 5))
	assert.Equal(
		[]*person{
			{Name: "Mark", Age: 50},
			{Name: "Stoney", Age: 49},
			{Name: "Matt", Age: 46},
			{Name: "Dillon", Age: 19},
			{Name: "Beth", Age: 54},
		},
		result,
	)
}

func TestNoGenerics(t *testing.T) {
	assert := assert.New(t)
	var strs []string
	oldConsumer := consume2.NewNoGenerics(
		consume2.Slice(consume2.AppendTo(&strs), 0, 2))
	for _, s := range []string{"hello", "world", "extra"} {
		if !oldConsumer.CanConsume() {
			break
		}
		oldConsumer.Consume(&s)
	}
	assert.Equal([]string{"hello", "world"}, strs)
}

func BenchmarkAppendTo(b *testing.B) {
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		var result []person
		consumer := consume2.AppendTo(&result)
		writePeopleInLoop(people[:], consume2.Slice(consumer, 0, 1000))
	}
}

func BenchmarkPagerFilterp(b *testing.B) {
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		pager := consume2.NewPageBuilder[person](17, 100)
		writePeopleInLoop(
			people[:],
			consume2.Filterp[person](
				pager,
				func(ptr *person) bool {
					ptr.Age *= 2
					return true
				},
			),
		)
		pager.Build()
	}
}

func BenchmarkPagerMapper(b *testing.B) {
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		pager := consume2.NewPageBuilder[person](17, 100)
		writePeopleInLoop(
			people[:],
			consume2.Map[person, person](
				pager,
				func(value person) person {
					value.Age *= 2
					return value
				},
			),
		)
		pager.Build()
	}
}

func BenchmarkPagerFilter(b *testing.B) {
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		pager := consume2.NewPageBuilder[person](17, 100)
		writePeopleInLoop(
			people[:],
			consume2.Filter[person](
				pager,
				func(value person) bool {
					return value.Name == "Beth"
				},
			),
		)
		pager.Build()
	}
}

func feedInts(t *testing.T, consumer consume2.Consumer[int]) {
	assert := assert.New(t)
	idx := 0
	for consumer.CanConsume() {
		consumer.Consume(idx)
		idx++
	}
	assert.Panics(func() {
		consumer.Consume(idx)
	})
}

func writePeopleInLoop(
	people []person, consumer consume2.Consumer[person]) {
	index := 0
	for consumer.CanConsume() {
		consumer.Consume(people[index%len(people)])
		index++
	}
}
