package addrconv

// Converter defines the strategy interface for address conversion.
// Each chain family implements this interface to provide its specific address conversion logic.
type Converter interface {
	// ConvertToBytes converts an address string to bytes according to the chain's format
	ConvertToBytes(address string) ([]byte, error)

	// Supports returns true if this converter supports the given chain family
	Supports(family string) bool
}
