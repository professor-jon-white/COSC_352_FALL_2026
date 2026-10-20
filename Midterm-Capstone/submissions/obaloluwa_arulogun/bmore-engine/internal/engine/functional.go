package engine

type Predicate[T any] func(T) bool

func Filter[T any](items []T, pred Predicate[T]) []T {
out := make([]T, 0, len(items)/4)
for i := range items {
if pred(items[i]) {
out = append(out, items[i])
}
}
return out
}

func MapSlice[T any, R any](items []T, fn func(T) R) []R {
out := make([]R, len(items))
for i := range items {
out[i] = fn(items[i])
}
return out
}

func Fold[T any, R any](items []T, initial R, folder func(R, T) R) R {
acc := initial
for i := range items {
acc = folder(acc, items[i])
}
return acc
}

func GroupBy[T any, K comparable](items []T, keyFn func(T) K) map[K][]T {
grouped := make(map[K][]T)
for i := range items {
k := keyFn(items[i])
grouped[k] = append(grouped[k], items[i])
}
return grouped
}
