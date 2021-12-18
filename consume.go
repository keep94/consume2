// Package consume2 provides ways to consume go values using generics.
package consume2

const (
	kCantConsume = "Can't consume"
)

// Consumer[T] consumes values of type T.
type Consumer[T any] interface {

	// CanConsume returns true if this instance can consume a value.
	// once CanConsume returns false, it should always return false.
	CanConsume() bool

	// Consume consumes a value. Consume panics if CanConsume returns false.
	Consume(value T)
}

// ConsumeFinalizer[T] adds a Finalize method to Consumer[T].
type ConsumeFinalizer[T any] interface {
	Consumer[T]

	// Caller must call Finalize after it is done passing values to this
	// Consumer. Once caller calls Finalize, CanConsume() returns false and
	// Consume() panics. Calls to Finalize are idempotent.
	Finalize()
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

// MustCanConsume[T] panics if c cannot consume.
func MustCanConsume[T any](c Consumer[T]) {
	if !c.CanConsume() {
		panic(kCantConsume)
	}
}

// AppendTo[T] returns a Consumer[T] that appends values to the slice
// pointed to by aSlicePtr. The CanConsume method of returned consumer
// always returns true.
func AppendTo[T any](aSlicePtr *[]T) Consumer[T] {
	return &appendConsumer[T]{slicePtr: aSlicePtr}
}

// AppendPtrsTo[T] returns a Consumer[T] that appends pointers to values
// to the slice pointed to by aSlicePtr. The CanConsume method of returned
// consumer always returns true.
func AppendPtrsTo[T any](aSlicePtr *[]*T) Consumer[T] {
	return &appendPtrConsumer[T]{slicePtr: aSlicePtr}
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
// filter function true onto the underlying consumer.
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
	switch len(filters) {
	case 0:
		return trueFunc[T]
	case 1:
		return filters[0]
	default:
		filterList := make([]func(T) bool, len(filters))
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
	switch len(consumers) {
	case 0:
		return nilConsumer[T]{}
	case 1:
		return consumers[0]
	default:
		consumerList := make([]Consumer[T], len(consumers))
		copy(consumerList, consumers)
		return &multiConsumer[T]{consumers: consumerList}
	}
}

// Page[T] returns a ConsumeFinalizer[T] that does pagination. The T values in
// the page fetched get stored in the slice pointed to by aSlicePtr.
// If there are more pages after page fetched, Page sets morePages to true;
// otherwise, it sets morePages to false. Note that the values stored at
// aSlicePtr and morePages are undefined until caller calls Finalize() on
// returned ConsumeFinalizer[T]. Page panics if zeroBasedPageNo is negative,
// or if itemsPerPage <= 0.
func Page[T any](
	zeroBasedPageNo int,
	itemsPerPage int,
	aSlicePtr *[]T,
	morePages *bool) ConsumeFinalizer[T] {
	if zeroBasedPageNo < 0 {
		panic("zeroBasedPageNo must be non-negative")
	}
	if itemsPerPage <= 0 {
		panic("itemsPerPage must be positive")
	}
	ensureEmptyWithCapacity(aSlicePtr, itemsPerPage+1)
	consumer := Slice(
		AppendTo(aSlicePtr),
		zeroBasedPageNo*itemsPerPage,
		(zeroBasedPageNo+1)*itemsPerPage+1)
	return &pageConsumer[T]{
		Consumer:     consumer,
		itemsPerPage: itemsPerPage,
		aSlicePtr:    aSlicePtr,
		morePages:    morePages}
}

// Nil[T] returns a Consumer[T] that consumes no T values. The CanConsume()
// method always returns false and the Consume() method always panics.
func Nil[T any]() Consumer[T] {
	return nilConsumer[T]{}
}

type appendConsumer[T any] struct {
	slicePtr *[]T
}

func (a *appendConsumer[T]) CanConsume() bool { return true }

func (a *appendConsumer[T]) Consume(value T) {
	*a.slicePtr = append(*a.slicePtr, value)
}

type appendPtrConsumer[T any] struct {
	slicePtr *[]*T
}

func (a *appendPtrConsumer[T]) CanConsume() bool { return true }

func (a *appendPtrConsumer[T]) Consume(value T) {
	*a.slicePtr = append(*a.slicePtr, &value)
}

type sliceConsumer[T any] struct {
	consumer Consumer[T]
	start    int
	end      int
	idx      int
}

func (s *sliceConsumer[T]) CanConsume() bool {
	return s.consumer.CanConsume() && s.idx < s.end
}

func (s *sliceConsumer[T]) Consume(value T) {
	MustCanConsume[T](s)
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
	MustCanConsume[T](f)
	if f.filter(value) {
		f.Consumer.Consume(value)
	}
}

type filterpConsumer[T any] struct {
	Consumer[T]
	filter func(ptr *T) bool
}

func (f *filterpConsumer[T]) Consume(value T) {
	MustCanConsume[T](f)
	if f.filter(&value) {
		f.Consumer.Consume(value)
	}
}

type mapConsumer[T, U any] struct {
	Consumer[U]
	mapper func(T) U
}

func (m *mapConsumer[T, U]) Consume(value T) {
	MustCanConsume[T](m)
	m.Consumer.Consume(m.mapper(value))
}

type maybeMapConsumer[T, U any] struct {
	Consumer[U]
	mapper func(T) (U, bool)
}

func (m *maybeMapConsumer[T, U]) Consume(value T) {
	MustCanConsume[T](m)
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
	MustCanConsume[T](m)
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

type pageConsumer[T any] struct {
	Consumer[T]
	itemsPerPage int
	aSlicePtr    *[]T
	morePages    *bool
	finalized    bool
}

func (p *pageConsumer[T]) Finalize() {
	if p.finalized {
		return
	}
	p.finalized = true
	p.Consumer = nilConsumer[T]{}
	if len(*p.aSlicePtr) == p.itemsPerPage+1 {
		*p.morePages = true
		*p.aSlicePtr = (*p.aSlicePtr)[:p.itemsPerPage]
	} else {
		*p.morePages = false
	}
}

func ensureEmptyWithCapacity[T any](aSlicePtr *[]T, capacity int) {
	if cap(*aSlicePtr) < capacity {
		*aSlicePtr = make([]T, 0, capacity)
	} else {
		*aSlicePtr = (*aSlicePtr)[:0]
	}
}

type nilConsumer[T any] struct {
}

func (n nilConsumer[T]) CanConsume() bool {
	return false
}

func (n nilConsumer[T]) Consume(value T) {
	panic(kCantConsume)
}

type takeWhileConsumer[T any] struct {
	consumer Consumer[T]
	filter   func(value T) bool
	done     bool
}

func (t *takeWhileConsumer[T]) CanConsume() bool {
	return t.consumer.CanConsume() && !t.done
}

func (t *takeWhileConsumer[T]) Consume(value T) {
	MustCanConsume[T](t)
	if !t.filter(value) {
		t.done = true
		return
	}
	t.consumer.Consume(value)
}

func trueFunc[T any](value T) bool {
	return true
}
