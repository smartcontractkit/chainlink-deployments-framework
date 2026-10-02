package statediff

import "errors"

// UnmarshalJSON requires an explicit kind rather than silently treating an
// omitted field as KindAdded's zero value. Decoding through DecodeJSON also
// preserves exact numbers in arbitrary before/after values, including when a
// caller uses encoding/json.Unmarshal directly on a Change.
func (c *Change) UnmarshalJSON(raw []byte) error {
	type changeJSON Change
	var decoded struct {
		changeJSON
		Kind *ChangeKind `json:"kind"`
	}
	if err := DecodeJSON(raw, &decoded); err != nil {
		return err
	}
	if decoded.Kind == nil {
		return errors.New("change kind is required and must not be null")
	}
	decoded.changeJSON.Kind = *decoded.Kind
	*c = Change(decoded.changeJSON)

	return nil
}
