package operations

import (
	"context"
	"errors"
	"sync"

	"github.com/Masterminds/semver/v3"
	"github.com/smartcontractkit/chainlink-common/pkg/logger"
)

// Bundle contains the dependencies required by Operations API and is passed to the OperationHandler and SequenceHandler.
// It contains the Logger, Reporter and the context.
// Use NewBundle to create a new Bundle.
type Bundle struct {
	Logger     logger.Logger
	GetContext func() context.Context
	reporter   Reporter
	// internal use only, for storing the hash of the report to avoid repeat sha256 computation.
	reportHashCache   *sync.Map
	OperationRegistry *OperationRegistry
}

// BundleOption is a functional option for configuring a Bundle
type BundleOption func(*Bundle)

// WithOperationRegistry sets a custom OperationRegistry for the Bundle
func WithOperationRegistry(registry *OperationRegistry) BundleOption {
	return func(b *Bundle) {
		b.OperationRegistry = registry
	}
}

// NewBundle creates and returns a new Bundle.
func NewBundle(getContext func() context.Context, logger logger.Logger, reporter Reporter, opts ...BundleOption) Bundle {
	b := Bundle{
		Logger:            logger,
		GetContext:        getContext,
		reporter:          reporter,
		reportHashCache:   &sync.Map{},
		OperationRegistry: NewOperationRegistry(),
	}

	// Apply all provided options
	for _, opt := range opts {
		opt(&b)
	}

	return b
}

// OperationHandler is the function signature of an operation handler.
type OperationHandler[IN, OUT, DEP any] func(e Bundle, deps DEP, input IN) (output OUT, err error)

// Definition is the metadata for a sequence or an operation.
// It contains the ID, version and description.
// This definition and OperationHandler together form the composite keys for an Operation.
// 2 Operations are considered the same if they have the Definition and OperationHandler.
type Definition struct {
	ID          string          `json:"id"`
	Version     *semver.Version `json:"version"`
	Description string          `json:"description"`
}

// Operation is the low level building blocks of the Operations API.
// Developers define their own operation with custom input and output types.
// Each operation should only perform max 1 side effect (e.g. send a transaction, post a job spec...)
// Use NewOperation to create a new operation.
type Operation[IN, OUT, DEP any] struct {
	def     Definition
	handler OperationHandler[IN, OUT, DEP]
}

// ID returns the operation ID.
func (o *Operation[IN, OUT, DEP]) ID() string {
	return o.def.ID
}

// Version returns the operation semver version in string.
func (o *Operation[IN, OUT, DEP]) Version() string {
	return o.def.Version.String()
}

// Description returns the operation description.
func (o *Operation[IN, OUT, DEP]) Description() string {
	return o.def.Description
}

// Def returns the operation definition.
func (o *Operation[IN, OUT, DEP]) Def() Definition {
	return o.def
}

// execute runs the operation by calling the OperationHandler.
func (o *Operation[IN, OUT, DEP]) execute(b Bundle, deps DEP, input IN) (output OUT, err error) {
	b.Logger.Infow("Executing operation",
		"id", o.def.ID, "version", o.def.Version, "description", o.def.Description)

	return o.handler(b, deps, input)
}

// AsUntyped converts the operation to an untyped operation.
// This is useful for storing operations in a slice or passing them around without type constraints.
// Warning: The input and output types will be converted to `any`, so type safety is lost.
func (o *Operation[IN, OUT, DEP]) AsUntyped() *Operation[any, any, any] {
	return &Operation[any, any, any]{
		def: o.def,
		handler: func(b Bundle, deps any, input any) (any, error) {
			var typedInput IN
			if input != nil {
				var ok bool
				if typedInput, ok = input.(IN); !ok {
					return nil, errors.New("input type mismatch")
				}
			}

			var typedDeps DEP
			if deps != nil {
				var ok bool
				if typedDeps, ok = deps.(DEP); !ok {
					return nil, errors.New("dependencies type mismatch")
				}
			}

			return o.handler(b, typedDeps, typedInput)
		},
	}
}

// NewOperation creates a new operation.
// Version can be created using semver.MustParse("1.0.0") or semver.New("1.0.0").
// Note: The handler should only perform maximum 1 side effect.
func NewOperation[IN, OUT, DEP any](
	id string, version *semver.Version, description string, handler OperationHandler[IN, OUT, DEP],
) *Operation[IN, OUT, DEP] {
	return &Operation[IN, OUT, DEP]{
		def: Definition{
			ID:          id,
			Version:     version,
			Description: description,
		},
		handler: handler,
	}
}

// EmptyInput is a placeholder for operations that do not require input.
type EmptyInput struct{}
