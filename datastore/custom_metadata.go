package datastore

// CustomMetadata is the interface we want to support.
type CustomMetadata interface {
	Clone() CustomMetadata
}
