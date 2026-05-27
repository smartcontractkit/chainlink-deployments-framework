package datastore

import "slices"

// deleteFromSlice deletes the first occurrence of item from the slice.
// If item is not found, the slice is returned unchanged.
func deleteFromSlice[T comparable](slice []T, item T) []T {
	if idx := slices.Index(slice, item); idx != -1 {
		return slices.Delete(slice, idx, idx+1)
	}

	return slice
}
