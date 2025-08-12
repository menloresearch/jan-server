package functional

func Map[T, V any](slice []T, f func(T) V) []V {
	result := make([]V, len(slice))
	for i, v := range slice {
		result[i] = f(v)
	}

	return result
}
