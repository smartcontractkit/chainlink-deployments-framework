package datastore

// TestMetadata is a struct that can be used as a default metadata type.
type TestMetadata struct {
	Data    string            `json:"data"`
	Version int               `json:"version"`
	Tags    []string          `json:"tags"`
	Extra   map[string]string `json:"extra"`
	Nested  NestedMeta        `json:"nested"`
}

type NestedMeta struct {
	Flag   bool   `json:"flag"`
	Detail string `json:"detail"`
}

// DefaultMetadata implements the Cloneable interface
func (d TestMetadata) Clone() CustomMetadata {
	extra := make(map[string]string, len(d.Extra))
	for k, v := range d.Extra {
		extra[k] = v
	}
	return TestMetadata{
		Data:    d.Data,
		Version: d.Version,
		Tags:    append([]string{}, d.Tags...),
		Extra:   extra,
		Nested:  d.Nested,
	}
}
