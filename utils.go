package main

type void struct{}

type Set[T comparable] map[T]void

func Iif[T any](c bool, t T, f T) T {
	if c {
		return t
	} else {
		return f
	}
}

func SumInt(vs ...int) (sum int) {
	for _, v := range vs {
		sum += v
	}
	return
}

func RemoveAt[T any](arr *[]T, index int) {
	*arr = append((*arr)[:index], (*arr)[index+1:]...)
}

func Duplicate[T any](src []T) []T {
	var dst = make([]T, len(src))
	copy(dst, src)
	return dst
}

type Equaler[T any] interface {
	Equal(T) bool
}

// O(n)
func IndexOf[T Equaler[T]](arr []*T, element *T) (index int, ok bool) {
	var item *T
	for index, item = range arr {
		if ok = (*element).Equal(*item); ok {
			return
		}
	}
	return
}

// O(n)
func Contains[T Equaler[T]](arr []*T, element *T) (ok bool) {
	_, ok = IndexOf(arr, element)
	return
}

// O(n * m) (excluding copy and remove)
func Intersect[T Equaler[T]](left []*T, right []*T) (result []*T) {
	for i := len(left) - 1; i >= 0; i-- {
		if Contains(right, left[i]) {
			result = append(result, left[i])
		}
	}
	return result
}

// O(n * m) (excluding copy and remove)
func Complement[T Equaler[T]](left []*T, right []*T) (result []*T) {
	for i := len(right) - 1; i >= 0; i-- {
		if Contains(left, right[i]) {
			result = append(result, right[i])
		}
	}
	return result
}

func Filter[T any](arr []T, check func(T) bool) (result []*T) {
	for _, item := range arr {
		if check(item) {
			result = append(result, &item)
		}
	}
	return result
}

func Without[T Equaler[T]](arr []*T, element *T) []*T {
	index, ok := IndexOf(arr, element)
	if !ok {
		return arr
	}
	return append(arr[:index], arr[index+1:]...)
}

func Map[T, R any](arr []*T, mapFn func(*T) *R) (result []*R) {
	result = make([]*R, len(arr))
	for index, item := range arr {
		result[index] = mapFn(item)
	}
	return result
}

func Assert(exp bool, message string) {
	if !exp {
		panic(message)
	}
}
