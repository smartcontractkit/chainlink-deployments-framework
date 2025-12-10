package memory

import (
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func errNotFoundID(entityType string, id string) error {
	return status.Errorf(codes.NotFound, "%s with id %s not found", entityType, id)
}

func errNilRequest() error {
	return status.Error(codes.InvalidArgument, "request cannot be nil")
}

func errUUIDLookupNotSupported() error {
	return status.Error(codes.InvalidArgument, "uuid lookup is deprecated and not supported by this implementation")
}
