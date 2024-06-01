package solver

type void struct{}

type set[T comparable] map[T]void

func (s set[T]) Intersect(x set[T]) []T {
	var result []T
	for k := range s {
		if _, ok := x[k]; ok {
			result = append(result, k)
		}
	}
	return result
}

func Intersect[T comparable](a, b []T) (result []T) {
	var hash = make(set[T])
	for _, v := range a {
		hash[v] = void{}

	}
	for _, v := range b {
		if _, ok := hash[v]; ok {
			result = append(result, v)
		}
	}
	return
}

func Complement[T comparable](a, b []T) (result []T) {
	var hash = make(set[T])
	for _, v := range a {
		hash[v] = void{}
	}
	for _, v := range b {
		if _, ok := hash[v]; !ok {
			result = append(result, v)
		}
	}
	return
}

func Copy[T any](a []T) []T {
	var result = make([]T, len(a))
	copy(result, a)
	return result
}
