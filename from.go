package consume2

// FromSlice sends values in aslice to consumer.
func FromSlice[T any](aslice []T, consumer Consumer[T]) {
	for index := 0; index < len(aslice) && consumer.CanConsume(); index++ {
		consumer.Consume(aslice[index])
	}
}

// FromPtrSlice sends values in aslice to consumer skipping nil pointers in
// aslice.
func FromPtrSlice[T any](aslice []*T, consumer Consumer[T]) {
	for index := 0; index < len(aslice) && consumer.CanConsume(); index++ {
		if aslice[index] == nil {
			continue
		}
		consumer.Consume(*aslice[index])
	}
}

// FromIntGenerator sends ints from generator to consumer. generator returns
// a negative number when there are no more ints to send.
func FromIntGenerator(generator func() int, consumer Consumer[int]) {
	for consumer.CanConsume() {
		value := generator()
		if value < 0 {
			break
		}
		consumer.Consume(value)
	}
}

// FromGenerator sends values from generator to consumer. generator returns
// false when there are no more values to send.
func FromGenerator[T any](generator func() (T, bool), consumer Consumer[T]) {
	for consumer.CanConsume() {
		value, ok := generator()
		if !ok {
			break
		}
		consumer.Consume(value)
	}
}
