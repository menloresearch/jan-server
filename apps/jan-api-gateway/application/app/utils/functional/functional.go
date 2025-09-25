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

func ConvertToMap[T, V comparable](slice []T, f func(T) V) map[V]T {
	result := make(map[V]T, len(slice))
	for _, v := range slice {
		key := f(v)
		result[key] = v
	}
	return result
}

func GetMapKeys[K comparable, V any](m map[K]V) []K {
	keys := make([]K, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	return keys
}
