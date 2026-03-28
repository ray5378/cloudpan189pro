package ptr

// Of 返回传入值的指针
func Of[T any](v T) *T {
	return &v
}

// Val 返回指针指向的值
func Val[T any](v *T) T {
	if v == nil {
		var zero T

		return zero
	}

	return *v
}
