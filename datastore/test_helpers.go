package datastore

// testMetadata is a custom metadata type for testing purposes.
type testMetadata struct {
	ChainSelector uint64 `json:"chain_selector"`
	Field         string `json:"field"`
}
