package datastore

import "encoding/json"

// DefaultMetadata is a struct that can be used as a default metadata type.
type DefaultMetadata struct {
	Data string `json:"data"`
}

// DefaultMetadata implements the Cloneable interface
func (dm DefaultMetadata) Clone() DefaultMetadata { return dm }

// MarshalJSON can handle both JSON and non-JSON data in the Data field.
// Without custom marshaling, the Data field would be double-encoded if it contains JSON.
func (dm DefaultMetadata) MarshalJSON() ([]byte, error) {
	// Try to parse the Data field as JSON
	var jsonData interface{}
	err := json.Unmarshal([]byte(dm.Data), &jsonData)

	if err == nil {
		// Valid JSON - Marshal the JSON data into a map prevents double-encoding
		return json.Marshal(map[string]interface{}{
			"data": jsonData,
		})
	}

	// Not valid JSON - Alias is used to avoid recursion
	type Alias DefaultMetadata

	return json.Marshal(Alias(dm))
}
