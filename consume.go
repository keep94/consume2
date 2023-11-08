// Package consume2 builds pipelines that consume values using Go generics.
package consume2

// Consumer[T] consumes values of type T.
type Consumer[T any] interface {

	// CanConsume returns true if this instance can consume a value.
	// Once CanConsume returns false, it should always return false.
	CanConsume() bool

	// Consume consumes a value. If CanConsume returns false, calling Consume
	// does not consume the value.
	Consume(value T)
}

// AsFunc converts a Consumer into a function that consumes its paramter
// and returns false when no more values can be consumed. AsFunc allows
// interoperability with other go packages such as github.com/google/btree.
func AsFunc[T any](consumer Consumer[T]) func(T) bool {
	return func(value T) bool {
		consumer.Consume(value)
		return consumer.CanConsume()
	}
}

// ConsumerFunc[T] makes any function accepting a T value implement
// Consumer[T]. CanConsume always returns true.
type ConsumerFunc[T any] func(value T)

// Consume invokes c, this function.
func (c ConsumerFunc[T]) Consume(value T) {
	c(value)
}

// CanConsume always returns true.
func (c ConsumerFunc[T]) CanConsume() bool {
	return true
}

// AppendTo[T] returns a Consumer[T] that appends values to the slice
// pointed to by aSlicePtr. The CanConsume method of returned consumer
// always returns true.
func AppendTo[T any](aSlicePtr *[]T) Consumer[T] {
	return (*appendConsumer[T])(aSlicePtr)
}

// AppendPtrsTo[T] returns a Consumer[T] that appends pointers to values
// to the slice pointed to by aSlicePtr. The CanConsume method of returned
// consumer always returns true.
func AppendPtrsTo[T any](aSlicePtr *[]*T) Consumer[T] {
	return (*appendPtrConsumer[T])(aSlicePtr)
}

// Slice[T] returns a Consumer[T] that passes the start th value consumed
// inclusive to the end th value consumed exclusive onto the underlying
// consumer where start and end are zero based. Note that if end <= start,
// the underlying consumer will never get any values. A negative start or end
// is treated as zero.
func Slice[T any](consumer Consumer[T], start, end int) Consumer[T] {
	return &sliceConsumer[T]{consumer: consumer, start: start, end: end}
}

// Filter[T] returns a Consumer[T] that passes only the values for which the
// filter function returns true onto the underlying consumer.
func Filter[T any](
	consumer Consumer[T], filter func(value T) bool) Consumer[T] {
	return &filterConsumer[T]{Consumer: consumer, filter: filter}
}

// Filterp[T] works like Filter[T] except that the filter function accepts
// *T instead of T. If the filter function mutates the T value via the pointer
// passed to it and returns true, the mutated value is sent to the underlying
// consumer.
func Filterp[T any](
	consumer Consumer[T], filter func(ptr *T) bool) Consumer[T] {
	return &filterpConsumer[T]{Consumer: consumer, filter: filter}
}

// TakeWhile[T] works like Filter[T] except that returned consumer only
// accepts values until one is filtered out.
func TakeWhile[T any](
	consumer Consumer[T], filter func(value T) bool) Consumer[T] {
	return &takeWhileConsumer[T]{consumer: consumer, filter: filter}
}

// ComposeFilters[T] returns a single function that filters T values by
// ANDing together all the filter functions passed in. The returned filter
// function applies the first function in filters then the second and so
// forth. If a function in filters returns false, then the functions after
// it are not evaluated. The returned filter function returns true for a
// value only if all the functions in filters return true for that value.
func ComposeFilters[T any](filters ...func(T) bool) func(T) bool {
	switch length := len(filters); length {
	case 0:
		return trueFunc[T]
	case 1:
		return filters[0]
	default:
		filterList := make([]func(T) bool, length)
		copy(filterList, filters)
		return func(value T) bool {
			for _, f := range filterList {
				if !f(value) {
					return false
				}
			}
			return true
		}
	}
}

// Map[T,U] returns a Consumer[T] that applies a mapper function to the T
// value being consumed and sends the resulting U value to the underlying
// consumer.
func Map[T, U any](
	consumer Consumer[U], mapper func(T) U) Consumer[T] {
	return &mapConsumer[T, U]{Consumer: consumer, mapper: mapper}
}

// MaybeMap[T,U] works like Map[T,U] except that the mapper function can
// return false for a T value in which case no corresponding U value is sent
// to the underlying consumer.
func MaybeMap[T, U any](
	consumer Consumer[U], mapper func(T) (U, bool)) Consumer[T] {
	return &maybeMapConsumer[T, U]{Consumer: consumer, mapper: mapper}
}

// Compose[T] returns all the Consumer[T] values passed to it as a single
// Consumer[T]. When returned consumer consumes a value, all the passed in
// consumers consume that same value. The CanConsume method of returned
// consumer returns false when the CanConsume method of all the passed in
// consumers returns false.
func Compose[T any](consumers ...Consumer[T]) Consumer[T] {
	switch length := len(consumers); length {
	case 0:
		return nilConsumer[T]{}
	case 1:
		return consumers[0]
	default:
		consumerList := make([]Consumer[T], length)
		copy(consumerList, consumers)
		return &multiConsumer[T]{consumers: consumerList}
	}
}

// PageBuilder[T] is a Consumer[T] that builds a specific page of T values.
// It consumes just enough T values needed to build the desired page.
type PageBuilder[T any] struct {
	consumer      Consumer[T]
	values        []T
	valuesPerPage int
}

// NewPageBuilder[T] creates a PageBuilder[T].
// NewPageBuilder[T] panics if zeroBasedPageNo is negative, or if
// valuesPerPage <= 0.
func NewPageBuilder[T any](
	zeroBasedPageNo int, valuesPerPage int) *PageBuilder[T] {
	if zeroBasedPageNo < 0 {
		panic("zeroBasedPageNo must be non-negative")
	}
	if valuesPerPage <= 0 {
		panic("valuesPerPage must be positive")
	}
	result := &PageBuilder[T]{
		valuesPerPage: valuesPerPage,
		values:        make([]T, 0, valuesPerPage+1),
	}
	result.consumer = Slice(
		AppendTo(&result.values),
		zeroBasedPageNo*valuesPerPage,
		(zeroBasedPageNo+1)*valuesPerPage+1)
	return result
}

// CanConsume returns false when this builder has all the T values it needs
// to build the desired page.
func (p *PageBuilder[T]) CanConsume() bool {
	return p.consumer.CanConsume()
}

// Consume consumes a single T value.
func (p *PageBuilder[T]) Consume(value T) {
	p.consumer.Consume(value)
}

// Build builds the desired page of T values. morePages is true if there
// are more pages after the desired page. Build is called after this
// builder has consumed its T values.
func (p *PageBuilder[T]) Build() (values []T, morePages bool) {
	length := len(p.values)
	if length > p.valuesPerPage {
		length = p.valuesPerPage
		morePages = true
	}
	values = make([]T, length)
	copy(values, p.values)
	return
}

// Nil[T] returns a Consumer[T] that consumes no T values. The CanConsume()
// method always returns false and the Consume() method does nothing.
func Nil[T any]() Consumer[T] {
	return nilConsumer[T]{}
}

type appendConsumer[T any] []T

func (a *appendConsumer[T]) CanConsume() bool { return true }

func (a *appendConsumer[T]) Consume(value T) {
	*a = append(*a, value)
}

type appendPtrConsumer[T any] []*T

func (a *appendPtrConsumer[T]) CanConsume() bool { return true }

func (a *appendPtrConsumer[T]) Consume(value T) {
	*a = append(*a, &value)
}

type sliceConsumer[T any] struct {
	consumer Consumer[T]
	start    int
	end      int
	idx      int
}

func (s *sliceConsumer[T]) CanConsume() bool {
	return s.idx < s.end && s.consumer.CanConsume()
}

func (s *sliceConsumer[T]) Consume(value T) {
	if s.idx >= s.end {
		return
	}
	if s.idx >= s.start {
		s.consumer.Consume(value)
	}
	s.idx++
}

type filterConsumer[T any] struct {
	Consumer[T]
	filter func(value T) bool
}

func (f *filterConsumer[T]) Consume(value T) {
	if f.filter(value) {
		f.Consumer.Consume(value)
	}
}

type filterpConsumer[T any] struct {
	Consumer[T]
	filter func(ptr *T) bool
}

func (f *filterpConsumer[T]) Consume(value T) {
	if f.filter(&value) {
		f.Consumer.Consume(value)
	}
}

type mapConsumer[T, U any] struct {
	Consumer[U]
	mapper func(T) U
}

func (m *mapConsumer[T, U]) Consume(value T) {
	m.Consumer.Consume(m.mapper(value))
}

type maybeMapConsumer[T, U any] struct {
	Consumer[U]
	mapper func(T) (U, bool)
}

func (m *maybeMapConsumer[T, U]) Consume(value T) {
	if mvalue, ok := m.mapper(value); ok {
		m.Consumer.Consume(mvalue)
	}
}

type multiConsumer[T any] struct {
	consumers []Consumer[T]
}

func (m *multiConsumer[T]) CanConsume() bool {
	m.filterFinished()
	return len(m.consumers) > 0
}

func (m *multiConsumer[T]) Consume(value T) {
	for _, consumer := range m.consumers {
		consumer.Consume(value)
	}
}

func (m *multiConsumer[T]) filterFinished() {
	idx := 0
	for i := range m.consumers {
		if m.consumers[i].CanConsume() {
			m.consumers[idx] = m.consumers[i]
			idx++
		}
	}
	for i := idx; i < len(m.consumers); i++ {
		m.consumers[i] = nil
	}
	m.consumers = m.consumers[0:idx]
}

type nilConsumer[T any] struct {
}

func (n nilConsumer[T]) CanConsume() bool {
	return false
}

func (n nilConsumer[T]) Consume(value T) {
}

type takeWhileConsumer[T any] struct {
	consumer Consumer[T]
	filter   func(value T) bool
	done     bool
}

func (t *takeWhileConsumer[T]) CanConsume() bool {
	return !t.done && t.consumer.CanConsume()
}

func (t *takeWhileConsumer[T]) Consume(value T) {
	if t.done {
		return
	}
	if !t.filter(value) {
		t.done = true
		return
	}
	t.consumer.Consume(value)
}

func trueFunc[T any](value T) bool {
	return true
}
