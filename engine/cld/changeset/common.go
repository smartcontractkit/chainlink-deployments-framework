package changeset

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"reflect"
	"strings"

	fresolvers "github.com/smartcontractkit/chainlink-deployments-framework/changeset/resolvers"
	fdeployment "github.com/smartcontractkit/chainlink-deployments-framework/deployment"
)

// Configurations holds options for a configured changeset
type Configurations struct {
	InputChainOverrides []uint64

	// Present only when the migration was wired with
	// Configure(...).WithConfigResolver(...)
	ConfigResolver fresolvers.ConfigResolver

	// InputType contains the reflect.Type of the input struct for this changeset
	// This is useful for tools that need to generate templates or analyze the expected input
	InputType reflect.Type
}

// internalChangeSet provides an opaque type, to force the usage of only the ChangeSetImpl
// for this purpose, but allowing the flexibility of an interface to work around the lack of covariant
// type-parameters.
type internalChangeSet interface {
	noop() // unexported function to prevent arbitrary structs from implementing ChangeSet.
	Apply(env fdeployment.Environment) (fdeployment.ChangesetOutput, error)
	Configurations() (Configurations, error)
}

type ChangeSet internalChangeSet

type ConfiguredChangeSet interface {
	ChangeSet
	ThenWith(postProcessor PostProcessor) PostProcessingChangeSet
}

// WrappedChangeSet simply wraps a fdeployment.ChangeSetV2 to use it in the fluent interface, which hosts
// the "With" function, so you can write `ConfigureLegacy(myChangeSet).With(aConfig)` in a typesafe way, and pass
// that into the ChangeSets.Add() function.
type WrappedChangeSet[C any] struct {
	operation fdeployment.ChangeSetV2[C]
}

// ConfigureLegacy begins a chain of functions that pairs a legacy (pure function) fdeployment.ChangeSet to a config,
// for registration as a migration.
//
// Deprecated: This wraps the deprecated fdeployment.ChangeSet. Should use fdeployment.ChangeSetV2
func ConfigureLegacy[C any](operation fdeployment.ChangeSet[C]) WrappedChangeSet[C] {
	return Configure[C](fdeployment.CreateLegacyChangeSet(operation))
}

// Configure begins a chain of functions that pairs a fdeployment.ChangeSetV2 to a config, for registration as a
// migration.
func Configure[C any](operation fdeployment.ChangeSetV2[C]) WrappedChangeSet[C] {
	return WrappedChangeSet[C]{operation: operation}
}

// With returns a fully configured changeset, which pairs a [fdeployment.ChangeSet] with its configuration. It also
// allows extensions, such as a PostProcessing function.
func (f WrappedChangeSet[C]) With(config C) ConfiguredChangeSet {
	return ChangeSetImpl[C]{changeset: f, configProvider: func() (C, error) { return config, nil }}
}

// inputObject is a JSON object with a "payload" field that contains the actual input data for a Durable Pipeline.
type TypedJSON struct {
	Payload        json.RawMessage `json:"payload"`
	ChainOverrides []uint64        `json:"chainOverrides"` // Optional field for chain overrides
}

// WithJSON returns a fully configured changeset, which pairs a [fdeployment.ChangeSet] with its configuration based
// a JSON input. It also allows extensions, such as a PostProcessing function.
// InputStr must be a JSON object with a "payload" field that contains the actual input data for a Durable Pipeline.
// Example:
//
//	{
//	  "payload": {
//	    "chainSelector": 123456789,
//	    "value": 100
//	  }
//	}
//
// Note: Prefer WithEnvInput for durable_pipelines.go
func (f WrappedChangeSet[C]) WithJSON(_ C, inputStr string) ConfiguredChangeSet {
	return ChangeSetImpl[C]{changeset: f, configProvider: func() (C, error) {
		var config C

		if inputStr == "" {
			return config, errors.New("input is empty")
		}

		var inputObject TypedJSON
		if err := json.Unmarshal([]byte(inputStr), &inputObject); err != nil {
			return config, fmt.Errorf("JSON must be in JSON format with 'payload' fields: %w", err)
		}

		// If payload is null, decode it as null (which will give zero value)
		// If payload is missing, return an error
		if len(inputObject.Payload) == 0 {
			return config, errors.New("'payload' field is required")
		}

		payloadDecoder := json.NewDecoder(strings.NewReader(string(inputObject.Payload)))
		payloadDecoder.DisallowUnknownFields()
		if err := payloadDecoder.Decode(&config); err != nil {
			return config, fmt.Errorf("failed to unmarshal payload: %w", err)
		}

		return config, nil
	},
		inputChainOverrides: func() ([]uint64, error) {
			return loadInputChainOverrides(inputStr)
		},
	}
}

// envInputOptions holds the options for WithEnvInput.
type envInputOptions[C any] struct {
	inputModifier func(c C) (C, error)
}

// EnvInputOption is a function that configures WithEnvInput.
type EnvInputOption[C any] func(options *envInputOptions[C])

// InputModifierFunc allows providing a custom function to update the input.
// The return value of the modifier function is used as the final input for the changeset.
func InputModifierFunc[C any](modifier func(c C) (C, error)) EnvInputOption[C] {
	return func(options *envInputOptions[C]) {
		options.inputModifier = modifier
	}
}

// WithEnvInput returns a fully configured changeset, which pairs a [fdeployment.ChangeSet] with its configuration based
// on the input defined in durable_pipelines/inputs for durable pipelines. It also allows extensions, such as a PostProcessing function.
// Options:
// - InputModifierFunc: allows providing a custom function to update the input.
func (f WrappedChangeSet[C]) WithEnvInput(opts ...EnvInputOption[C]) ConfiguredChangeSet {
	options := &envInputOptions[C]{}
	for _, opt := range opts {
		opt(options)
	}

	inputStr := os.Getenv("DURABLE_PIPELINE_INPUT")

	return ChangeSetImpl[C]{changeset: f, configProvider: func() (C, error) {
		var config C

		if inputStr == "" {
			return config, errors.New("input is empty")
		}

		var inputObject TypedJSON
		if err := json.Unmarshal([]byte(inputStr), &inputObject); err != nil {
			return config, fmt.Errorf("JSON must be in JSON format with 'payload' fields: %w", err)
		}

		// If payload is null, decode it as null (which will give zero value)
		// If payload is missing, return an error
		if len(inputObject.Payload) == 0 {
			return config, errors.New("'payload' field is required")
		}

		payloadDecoder := json.NewDecoder(strings.NewReader(string(inputObject.Payload)))
		payloadDecoder.DisallowUnknownFields()
		if err := payloadDecoder.Decode(&config); err != nil {
			return config, fmt.Errorf("failed to unmarshal payload: %w", err)
		}

		if options.inputModifier != nil {
			conf, err := options.inputModifier(config)
			if err != nil {
				return conf, fmt.Errorf("failed to apply input modifier: %w", err)
			}

			return conf, nil
		}

		return config, nil
	},
		inputChainOverrides: func() ([]uint64, error) {
			return loadInputChainOverrides(inputStr)
		},
	}
}

func loadInputChainOverrides(inputStr string) ([]uint64, error) {
	if inputStr == "" {
		return nil, nil
	}

	// looks at the input JSON for "ChainOverrides" field
	var inputObject TypedJSON
	if err := json.Unmarshal([]byte(inputStr), &inputObject); err != nil {
		return nil, err
	}

	return inputObject.ChainOverrides, nil
}

// WithConfigFrom takes a provider function which returns a config or an error, and stores the error (if any) in
// the configured changeset. This is then used to abort execution with a cleaner message than a panic.
// This allows for more robust error handling to happen for complex configs, factored into a function, without
// needing to use the "mustGetConfig" pattern
func (f WrappedChangeSet[C]) WithConfigFrom(configProvider func() (C, error)) ConfiguredChangeSet {
	return ChangeSetImpl[C]{changeset: f, configProvider: configProvider}
}

// WithConfigResolver uses a registered config resolver to generate the configuration.
// It reads input from the DURABLE_PIPELINE_INPUT environment variable (JSON format)
// and uses the specified resolver to generate the typed configuration.
func (f WrappedChangeSet[C]) WithConfigResolver(resolver fresolvers.ConfigResolver) ConfiguredChangeSet {
	// Read input from environment variable
	inputStr := os.Getenv("DURABLE_PIPELINE_INPUT")

	configProvider := func() (C, error) {
		var zero C

		if inputStr == "" {
			return zero, errors.New("input is empty")
		}

		// Parse JSON input
		var inputObject TypedJSON
		if err := json.Unmarshal([]byte(inputStr), &inputObject); err != nil {
			return zero, fmt.Errorf("failed to parse resolver input as JSON: %w", err)
		}

		// If payload is null, pass it to the resolver (which will receive null)
		// If payload field is missing, return an error
		if len(inputObject.Payload) == 0 {
			return zero, errors.New("'payload' field is required")
		}

		// Call resolver – automatically unmarshal into its expected input type.
		typedConfig, err := fresolvers.CallResolver[C](resolver, inputObject.Payload)
		if err != nil {
			return zero, fmt.Errorf("config resolver failed: %w", err)
		}

		return typedConfig, nil
	}

	return ChangeSetImpl[C]{changeset: f, configProvider: configProvider,
		ConfigResolver: resolver,
		inputChainOverrides: func() ([]uint64, error) {
			return loadInputChainOverrides(inputStr)
		},
	}
}

var _ ConfiguredChangeSet = ChangeSetImpl[any]{}

type ChangeSetImpl[C any] struct {
	changeset           WrappedChangeSet[C]
	configProvider      func() (C, error)
	inputChainOverrides func() ([]uint64, error)

	// Present only when the migration was wired with
	// Configure(...).WithConfigResolver(...)
	ConfigResolver fresolvers.ConfigResolver
}

func (ccs ChangeSetImpl[C]) noop() {}

func (ccs ChangeSetImpl[C]) Apply(env fdeployment.Environment) (fdeployment.ChangesetOutput, error) {
	c, err := ccs.configProvider()
	if err != nil {
		return fdeployment.ChangesetOutput{}, err
	}
	err = ccs.changeset.operation.VerifyPreconditions(env, c)
	if err != nil {
		return fdeployment.ChangesetOutput{}, err
	}

	return ccs.changeset.operation.Apply(env, c)
}

func (ccs ChangeSetImpl[C]) Configurations() (Configurations, error) {
	var chainOverrides []uint64
	var err error

	if ccs.inputChainOverrides != nil {
		chainOverrides, err = ccs.inputChainOverrides()
		if err != nil {
			return Configurations{}, err
		}
	}

	// Get the type of C (the input struct type)
	var zero C
	inputType := reflect.TypeOf(zero)

	return Configurations{
		InputChainOverrides: chainOverrides,
		ConfigResolver:      ccs.ConfigResolver,
		InputType:           inputType,
	}, nil
}

// ThenWith adds post-processing to a configured changeset
func (ccs ChangeSetImpl[C]) ThenWith(postProcessor PostProcessor) PostProcessingChangeSet {
	return PostProcessingChangeSetImpl[C]{
		changeset:     ccs,
		postProcessor: postProcessor,
	}
}
