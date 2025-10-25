package memory

import "github.com/segmentio/ksuid"

// newProposalID generates a new proposal ID.
func newProposalID() string {
	return newID("prop")
}

// newJobID generates a new job ID.
func newJobID() string {
	return newID("job")
}

// newNodeID generates a new node ID.
func newNodeID() string {
	return newID("node")
}

// newID generates a new ID with a given prefix.
//
// This uses ksuid to generate a unique ID which differs from the Job Distributor ID format, to
// better differentiate from the UUID that are set on the job. Each ID should be prefixed with a
// string to identify the type of ID.
func newID(prefix string) string {
	return prefix + "_" + ksuid.New().String()
}
