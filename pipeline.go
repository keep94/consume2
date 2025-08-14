package consume2

// Pipeline is an abstraction that emits a group of U values from a group
// of T values. The returned Consumer consumes the T values and sends the
// U values to inner.
type Pipeline[T, U any] func(inner Consumer[U]) Consumer[T]

// Run returns a Consumer that collects the T values for this pipeline
// and sends the U values this pipeline emits to the consumer parameter.
func (p Pipeline[T, U]) Run(consumer Consumer[U]) Consumer[T] {
	return p(consumer)
}

// Call works like Run except that it calls f on each U value this
// pipeline emits.
func (p Pipeline[T, U]) Call(f func(value U)) Consumer[T] {
	return p.Run(Call(f))
}

// AppendTo returns a Consumer that collects the T values for this pipeline
// and appends the U values this pipeline emits to aSlicePtr.
func (p Pipeline[T, U]) AppendTo(aSlicePtr *[]U) Consumer[T] {
	return p.Run(AppendTo(aSlicePtr))
}

// AppendPtrsTo returns a Consumer that collects the T values for this
// pipeline and appends the pointers to the U values this pipeline emits
// to aSlicePtr.
func (p Pipeline[T, U]) AppendPtrsTo(aSlicePtr *[]*U) Consumer[T] {
	return p.Run(AppendPtrsTo(aSlicePtr))
}

// Join joins two pipelines into a single pipeline.
func Join[T, U, V any](
	first Pipeline[T, U], second Pipeline[U, V]) Pipeline[T, V] {
	return func(inner Consumer[V]) Consumer[T] {
		return first(second(inner))
	}
}

// PFilter returns a Pipeline that applies filter to the T values it receives
// and emits only those T values for which filter returns true.
func PFilter[T any](filter func(value T) bool) Pipeline[T, T] {
	return func(inner Consumer[T]) Consumer[T] {
		return Filter(inner, filter)
	}
}

// PFilterp is like PFilter except that the returned pipeline can mutate the
// T values it emits while leaving the original T values the same.
func PFilterp[T any](filter func(ptr *T) bool) Pipeline[T, T] {
	return func(inner Consumer[T]) Consumer[T] {
		return Filterp(inner, filter)
	}
}

// PMap returns a Pipeline that applies mapper to the T values it receives
// and emits the resulting U values.
func PMap[T, U any](mapper func(T) U) Pipeline[T, U] {
	return func(inner Consumer[U]) Consumer[T] {
		return Map(inner, mapper)
	}
}

// PMaybeMap returns a Pipeline that applies mapper to the T values it receives
// and emits the resulting U values for which mapper returns true.
func PMaybeMap[T, U any](mapper func(T) (U, bool)) Pipeline[T, U] {
	return func(inner Consumer[U]) Consumer[T] {
		return MaybeMap(inner, mapper)
	}
}

// PSlice returns a Pipeline that emits the start T value it receives
// inclusive up to the end T value it receives exclusive. start and end are
// zero based.
func PSlice[T any](start, end int) Pipeline[T, T] {
	return func(inner Consumer[T]) Consumer[T] {
		return Slice(inner, start, end)
	}
}

// PTakeWhile returns a Pipeline that emits the first T values it receives
// for which filter returns true.
func PTakeWhile[T any](filter func(value T) bool) Pipeline[T, T] {
	return func(inner Consumer[T]) Consumer[T] {
		return TakeWhile(inner, filter)
	}
}

// Identity returns a Pipeline that emits the same T values it receives.
func Identity[T any]() Pipeline[T, T] {
	return func(inner Consumer[T]) Consumer[T] {
		return inner
	}
}
