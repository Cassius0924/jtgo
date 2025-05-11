package ds

// Flag 标志位
type Flag[T ~int | ~int8 | ~int16 | ~int32 | ~int64] struct {
	value T
}

func NewFlag[T ~int | ~int8 | ~int16 | ~int32 | ~int64]() *Flag[T] {
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

func (f *Flag[T]) HasAny(flags ...T) bool {
	if len(flags) == 0 {
		return false
	}
	for _, flag := range flags {
		if f.value&flag != 0 {
			return true
		}
	}
	return false
}

func (f *Flag[T]) HasAll(flags ...T) bool {
	if len(flags) == 0 {
		return true
	}
	var combined T
	for _, flag := range flags {
		combined |= flag
	}
	return f.value&combined == combined
}
