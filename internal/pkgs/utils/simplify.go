package utils

func UseSimplify[T any](defaultValue T, vals ...T) T {
	if len(vals) == 0 {
		return defaultValue
	}

	return vals[0]
}
