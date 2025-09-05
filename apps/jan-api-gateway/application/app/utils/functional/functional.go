package functional

func Map[T, V any](slice []T, f func(T) V) []V {
	result := make([]V, len(slice))
	for i, v := range slice {
		result[i] = f(v)
	}

	return result
}

func Distinct[T comparable](slice []T) []T {
	seen := make(map[T]struct{})
	result := []T{}

	for _, v := range slice {
		if _, ok := seen[v]; !ok {
			result = append(result, v)
			seen[v] = struct{}{}
		}
	}
	return result
}
