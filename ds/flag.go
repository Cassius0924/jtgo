package ds

// Flag 标志位
type Flag[T ~int8] struct {
	value T
}

func NewFlag[T ~int8]() *Flag[T] {
	return &Flag[T]{}
}

func (f *Flag[T]) Set(flag T) {
	f.value |= flag
}

func (f *Flag[T]) Unset(flag T) {
	f.value &= ^flag
}

func (f *Flag[T]) Has(flag T) bool {
	return f.value&flag != 0
}

func (f *Flag[T]) Toggle(flag T) {
	f.value ^= flag
}

func (f *Flag[T]) Reset() {
	f.value = 0
}
