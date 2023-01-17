package hap

import "github.com/jinzhu/copier"

// DeepCopy returns a deep copy of T.
func DeepCopy[T any](t *T) T {
	var copy T
	copier.Copy(&copy, t)
	return copy
}
