package datastore

import (
	"encoding/json"
	"maps"
	"slices"
	"strings"
)

// LabelSet represents a set of labels on an address book entry.
type LabelSet struct {
	elements map[string]struct{}
}

// NewLabelSet initializes a new LabelSet with any number of labels.
func NewLabelSet(labels ...string) LabelSet {
	set := make(map[string]struct{}, len(labels))
	for _, l := range labels {
		set[l] = struct{}{}
	}

	return LabelSet{
		elements: set,
	}
}

// Add inserts one or more labels into the set.
func (s *LabelSet) Add(labels ...string) {
	if s.elements == nil {
		s.elements = make(map[string]struct{})
	}
	for _, l := range labels {
		s.elements[l] = struct{}{}
	}
}

// Remove deletes a label from the set, if it exists.
func (s *LabelSet) Remove(label string) {
	delete(s.elements, label)
}

// Contains checks if the set contains the given label.
func (s *LabelSet) Contains(label string) bool {
	_, ok := s.elements[label]

	return ok
}

// String returns the labels as a sorted, space-separated string.
//
// Implements the fmt.Stringer interface.
func (s *LabelSet) String() string {
	labels := s.List()
	if len(labels) == 0 {
		return ""
	}

	// Concatenate the sorted labels into a single string
	return strings.Join(labels, " ")
}

// List returns the labels as a sorted slice of strings.
func (s *LabelSet) List() []string {
	if len(s.elements) == 0 {
		return []string{}
	}

	// Collect labels into a slice
	labels := slices.Collect(maps.Keys(s.elements))

	// Sort the labels to ensure consistent ordering
	slices.Sort(labels)

	return labels
}

// Equal checks if two LabelSets are equal.
func (s *LabelSet) Equal(other LabelSet) bool {
	return maps.Equal(s.elements, other.elements)
}

// Length returns the number of labels in the set.
func (s *LabelSet) Length() int {
	return len(s.elements)
}

// IsEmpty checks if the LabelSet is empty.
func (s *LabelSet) IsEmpty() bool {
	return s.Length() == 0
}

// Clone creates a copy of the LabelSet.
func (s *LabelSet) Clone() LabelSet {
	return LabelSet{
		elements: maps.Clone(s.elements),
	}
}

// MarshalJSON marshals the LabelSet as a JSON array of strings.
//
// Implements the json.Marshaler interface.
func (s LabelSet) MarshalJSON() ([]byte, error) {
	return json.Marshal(s.List())
}

// UnmarshalJSON unmarshals a JSON array of strings into the LabelSet.
//
// Implements the json.Unmarshaler interface.
func (s *LabelSet) UnmarshalJSON(data []byte) error {
	var labels []string
	if err := json.Unmarshal(data, &labels); err != nil {
		return err
	}

	// Initialize the LabelSet with the unmarshaled labels
	*s = NewLabelSet(labels...)

	return nil
}
