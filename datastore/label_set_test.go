package datastore

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func Test_NewLabelSet(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		give []string
		want []string
	}{
		{
			name: "no labels",
			give: []string{},
			want: []string{},
		},
		{
			name: "some labels",
			give: []string{"foo", "bar"},
			want: []string{"foo", "bar"},
		},
		{
			name: "non unique labels",
			give: []string{"foo", "bar", "foo"},
			want: []string{"foo", "bar"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			got := NewLabelSet(tt.give...)

			require.Equal(t, len(tt.want), got.Length())
			for _, l := range tt.want {
				assert.True(t, got.Contains(l), "expected label '%s' in the set", l)
			}
		})
	}
}

func Test_LabelSet_Add(t *testing.T) {
	t.Parallel()

	labels := NewLabelSet("initial")
	labels.Add("new")

	require.Equal(t, 2, labels.Length())
	assert.True(t, labels.Contains("initial"))
	assert.True(t, labels.Contains("new"))

	// Add duplicate "new" again; size should remain 2
	labels.Add("new")
	require.Equal(t, 2, labels.Length())

	t.Run("Add to nil elements LabelSet", func(t *testing.T) {
		t.Parallel()

		var ls LabelSet
		ls.Add("foo")

		require.Equal(t, 1, ls.Length())
		assert.True(t, ls.Contains("foo"))
	})
}

func Test_LabelSet_Remove(t *testing.T) {
	t.Parallel()

	labels := NewLabelSet("remove_me", "keep")
	labels.Remove("remove_me")

	require.Equal(t, 1, labels.Length())
	assert.False(t, labels.Contains("remove_me"))
	assert.True(t, labels.Contains("keep"))

	// Removing a non-existent item shouldn't change the size
	labels.Remove("non_existent")
	require.Equal(t, 1, labels.Length())
}

func Test_LabelSet_Contains(t *testing.T) {
	t.Parallel()

	got := NewLabelSet("foo", "bar")

	assert.True(t, got.Contains("foo"))
	assert.True(t, got.Contains("bar"))
	assert.False(t, got.Contains("baz"))
}

func Test_LabelSet_String(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name   string
		labels LabelSet
		want   string
	}{
		{
			name:   "Empty LabelSet",
			labels: NewLabelSet(),
			want:   "",
		},
		{
			name:   "Single label",
			labels: NewLabelSet("alpha"),
			want:   "alpha",
		},
		{
			name:   "Multiple labels in random order",
			labels: NewLabelSet("beta", "gamma", "alpha"),
			want:   "alpha beta gamma",
		},
		{
			name:   "Labels with special characters",
			labels: NewLabelSet("beta", "gamma!", "@alpha"),
			want:   "@alpha beta gamma!",
		},
		{
			name:   "Labels with spaces",
			labels: NewLabelSet("beta", "gamma delta", "alpha"),
			want:   "alpha beta gamma delta",
		},
		{
			name:   "Labels added in different orders",
			labels: NewLabelSet("delta", "beta", "alpha"),
			want:   "alpha beta delta",
		},
		{
			name:   "Labels with duplicate additions",
			labels: NewLabelSet("alpha", "beta", "alpha", "gamma", "beta"),
			want:   "alpha beta gamma",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			got := tt.labels.String()
			assert.Equal(t, tt.want, got, "LabelSet.String() should return the expected sorted string")
		})
	}
}

func Test_LabelSet_List(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		give []string
		want []string
	}{
		{
			name: "list with items",
			give: []string{"foo", "bar", "baz"},
			want: []string{"bar", "baz", "foo"},
		},
		{
			name: "empty list",
			give: []string{},
			want: []string{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			ms := NewLabelSet(tt.give...)
			got := ms.List()

			assert.Len(t, got, len(tt.want), "unexpected number of labels in the list")
			assert.ElementsMatch(t, tt.want, got, "unexpected labels in the list")
		})
	}
}

func Test_LabelSet_Equal(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		set1 LabelSet
		set2 LabelSet
		want bool
	}{
		{
			name: "Both sets empty",
			set1: NewLabelSet(),
			set2: NewLabelSet(),
			want: true,
		},
		{
			name: "First set empty, second set non-empty",
			set1: NewLabelSet(),
			set2: NewLabelSet("label1"),
			want: false,
		},
		{
			name: "First set non-empty, second set empty",
			set1: NewLabelSet("label1"),
			set2: NewLabelSet(),
			want: false,
		},
		{
			name: "Identical sets with single label",
			set1: NewLabelSet("label1"),
			set2: NewLabelSet("label1"),
			want: true,
		},
		{
			name: "Identical sets with multiple labels",
			set1: NewLabelSet("label1", "label2", "label3"),
			set2: NewLabelSet("label3", "label2", "label1"), // Different order
			want: true,
		},
		{
			name: "Different sets, same size",
			set1: NewLabelSet("label1", "label2", "label3"),
			set2: NewLabelSet("label1", "label2", "label4"),
			want: false,
		},
		{
			name: "Different sets, different sizes",
			set1: NewLabelSet("label1", "label2"),
			set2: NewLabelSet("label1", "label2", "label3"),
			want: false,
		},
		{
			name: "Subset sets",
			set1: NewLabelSet("label1", "label2"),
			set2: NewLabelSet("label1", "label2", "label3"),
			want: false,
		},
		{
			name: "Disjoint sets",
			set1: NewLabelSet("label1", "label2"),
			set2: NewLabelSet("label3", "label4"),
			want: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			got := tt.set1.Equal(tt.set2)
			assert.Equal(t, tt.want, got, "Equal(%v, %v) should be %v", tt.set1, tt.set2, tt.want)
		})
	}
}

func Test_LabelSet_Length(t *testing.T) {
	t.Parallel()

	got := NewLabelSet("foo", "bar", "baz")
	require.Equal(t, 3, got.Length())
}

func Test_LabelSet_IsEmpty(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		give LabelSet
		want bool
	}{
		{
			name: "Empty set",
			give: NewLabelSet(),
			want: true,
		},
		{
			name: "Non-empty set",
			give: NewLabelSet("foo"),
			want: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			got := tt.give.IsEmpty()
			assert.Equal(t, tt.want, got, "IsEmpty() should return %v", tt.want)
		})
	}
}

func Test_LabelSet_Clone(t *testing.T) {
	t.Run("Clone non-empty", func(t *testing.T) {
		t.Parallel()

		original := NewLabelSet("foo", "bar", "baz")
		clone := original.Clone()

		assert.Equal(t, original, clone, "Clone() should return an equal LabelSet")
		assert.NotSame(t, &original, &clone, "Clone() should return a different LabelSet instance")

		clone.Add("new")
		assert.NotEqual(t, original, clone, "Modifying the clone should not affect the original")
	})

	t.Run("Clone empty", func(t *testing.T) {
		t.Parallel()
		empty := NewLabelSet()
		cloned := empty.Clone()

		assert.Equal(t, empty, cloned, "Cloning an empty LabelSet should return an equal empty LabelSet")
		assert.NotSame(t, &empty, &cloned, "Cloned empty LabelSet should not be the same reference")
	})
}

func Test_LabelSet_MarshalJSON(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		give LabelSet
		want string
	}{
		{
			name: "Empty set",
			give: NewLabelSet(),
			want: `[]`,
		},
		{
			name: "Single label",
			give: NewLabelSet("foo"),
			want: `["foo"]`,
		},
		{
			name: "Multiple labels",
			give: NewLabelSet("foo", "bar"),
			want: `["bar","foo"]`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			got, err := json.Marshal(&tt.give)
			require.NoError(t, err)
			assert.JSONEq(t, tt.want, string(got))
		})
	}
}

func Test_LabelSet_UnmarshalJSON(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		give string
		want LabelSet
	}{
		{
			name: "Empty set",
			give: `[]`,
			want: NewLabelSet(),
		},
		{
			name: "Single label",
			give: `["foo"]`,
			want: NewLabelSet("foo"),
		},
		{
			name: "Multiple labels",
			give: `["foo", "bar"]`,
			want: NewLabelSet("bar", "foo"),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			var got LabelSet
			err := json.Unmarshal([]byte(tt.give), &got)
			require.NoError(t, err)
			assert.Equal(t, tt.want, got)
		})
	}
}
