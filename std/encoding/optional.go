package encoding

type Optional[T any] struct {
	value T
	isSet bool
}

func (o *Optional[T]) IsSet() bool {
	return o.isSet
}

func (o *Optional[T]) Set(v T) {
	o.value = v
	o.isSet = true
}

func (o *Optional[T]) Unset() {
	o.isSet = false
}

func (o *Optional[T]) Get() (T, bool) {
	return o.value, o.isSet
}

func (o *Optional[T]) GetOr(def T) T {
	if o.isSet {
		return o.value
	}
	return def
}

func (o *Optional[T]) Unwrap() T {
	if o.isSet {
		return o.value
	}
	panic("Optional value is not set")
}

// Deprecated: Will be removed in the future.
func (o *Optional[T]) Ptr() *T {
	if o.isSet {
		return &o.value
	}
	return nil
}

func Some[T any](v T) Optional[T] {
	return Optional[T]{value: v, isSet: true}
}

func None[T any]() Optional[T] {
	return Optional[T]{isSet: false}
}

// Deprecated: Will be removed in the future.
func OptionPtr[T any](v *T) Optional[T] {
	if v == nil {
		return Optional[T]{isSet: false}
	}
	return Optional[T]{value: *v, isSet: true}
}
