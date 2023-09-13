package utils

import (
	"github.com/linkbase/middleware"
	"sync"
)

type UniqueID = middleware.UniqueID

type UniqueSet = Set[UniqueID]

func NewUniqueSet(ids ...UniqueID) UniqueSet {
	set := make(UniqueSet)
	set.Insert(ids...)
	return set
}

type Set[T comparable] map[T]struct{}

func NewSet[T comparable](elements ...T) Set[T] {
	set := make(Set[T])
	set.Insert(elements...)
	return set
}

// Insert elements into the set,
// do nothing if the id existed
func (set Set[T]) Insert(elements ...T) {
	for i := range elements {
		set[elements[i]] = struct{}{}
	}
}

// Intersection returns the intersection with the given set
func (set Set[T]) Intersection(other Set[T]) Set[T] {
	ret := NewSet[T]()
	for elem := range set {
		if other.Contain(elem) {
			ret.Insert(elem)
		}
	}
	return ret
}

// Union returns the union with the given set
func (set Set[T]) Union(other Set[T]) Set[T] {
	ret := NewSet(set.Collect()...)
	ret.Insert(other.Collect()...)
	return ret
}

// Complement returns the complement with the given set
func (set Set[T]) Complement(other Set[T]) Set[T] {
	ret := NewSet(set.Collect()...)
	ret.Remove(other.Collect()...)
	return ret
}

// Check whether the elements exist
func (set Set[T]) Contain(elements ...T) bool {
	for i := range elements {
		_, ok := set[elements[i]]
		if !ok {
			return false
		}
	}
	return true
}

// Remove elements from the set,
// do nothing if set is nil or id not exists
func (set Set[T]) Remove(elements ...T) {
	for i := range elements {
		delete(set, elements[i])
	}
}

func (set Set[T]) Clear() {
	set.Remove(set.Collect()...)
}

// Get all elements in the set
func (set Set[T]) Collect() []T {
	elements := make([]T, 0, len(set))
	for elem := range set {
		elements = append(elements, elem)
	}
	return elements
}

// Len returns the number of elements in the set
func (set Set[T]) Len() int {
	return len(set)
}

type ConcurrentSet[T comparable] struct {
	inner sync.Map
}

func NewConcurrentSet[T comparable]() *ConcurrentSet[T] {
	return &ConcurrentSet[T]{}
}

// Insert elements into the set,
// do nothing if the id existed
func (set *ConcurrentSet[T]) Upsert(elements ...T) {
	for i := range elements {
		set.inner.Store(elements[i], struct{}{})
	}
}

func (set *ConcurrentSet[T]) Insert(element T) bool {
	_, exist := set.inner.LoadOrStore(element, struct{}{})
	return !exist
}

// Check whether the elements exist
func (set *ConcurrentSet[T]) Contain(elements ...T) bool {
	for i := range elements {
		_, ok := set.inner.Load(elements[i])
		if !ok {
			return false
		}
	}
	return true
}

// Remove elements from the set,
// do nothing if set is nil or id not exists
func (set *ConcurrentSet[T]) Remove(elements ...T) {
	for i := range elements {
		set.inner.Delete(elements[i])
	}
}

// Try remove element from set,
// return false if not exist
func (set *ConcurrentSet[T]) TryRemove(element T) bool {
	_, exist := set.inner.LoadAndDelete(element)
	return exist
}

// Get all elements in the set
func (set *ConcurrentSet[T]) Collect() []T {
	elements := make([]T, 0)
	set.inner.Range(func(key, value any) bool {
		elements = append(elements, key.(T))
		return true
	})
	return elements
}

func (set *ConcurrentSet[T]) Range(f func(element T) bool) {
	set.inner.Range(func(key, value any) bool {
		trueKey := key.(T)
		return f(trueKey)
	})
}
