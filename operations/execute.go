package operations

import (
	"errors"
	"fmt"

	"github.com/avast/retry-go/v4"
)

var ErrNotSerializable = errors.New("data cannot be safely written to disk without data lost, " +
	"avoid type that can't be serialized")

// ExecuteConfig is the configuration for the ExecuteOperation function.
type ExecuteConfig[IN, DEP any] struct {
	retryConfig RetryConfig[IN, DEP]
}

type ExecuteOption[IN, DEP any] func(*ExecuteConfig[IN, DEP])

type RetryConfig[IN, DEP any] struct {
	// Enabled determines if the retry is enabled for the operation.
	Enabled bool

	// Policy is the retry policy to control the behavior of the retry.
	Policy RetryPolicy

	// InputHook is a function that returns an updated input before retrying the operation.
	// The operation when retried will use the input returned by this function.
	// This is useful for scenarios like updating the gas limit.
	InputHook func(attempt uint, err error, input IN, deps DEP) IN
}

// newDisabledRetryConfig returns a default retry configuration that is initially disabled.
func newDisabledRetryConfig[IN, DEP any]() RetryConfig[IN, DEP] {
	return RetryConfig[IN, DEP]{
		Enabled: false,
		Policy: RetryPolicy{
			MaxAttempts: 10,
		},
	}
}

// RetryPolicy defines the arguments to control the retry behavior.
type RetryPolicy struct {
	MaxAttempts uint
}

// options returns the 'avast/retry' functional options for the retry policy.
func (p RetryPolicy) options() []retry.Option {
	return []retry.Option{
		retry.Attempts(p.MaxAttempts),
	}
}

// WithRetry is an ExecuteOption that enables the default retry for the operation.
func WithRetry[IN, DEP any]() ExecuteOption[IN, DEP] {
	return func(c *ExecuteConfig[IN, DEP]) {
		c.retryConfig.Enabled = true
	}
}

// WithRetryInput is an ExecuteOption that enables the default retry and provide an input
// transform function which will modify the input on each retry attempt.
func WithRetryInput[IN, DEP any](inputHookFunc func(uint, error, IN, DEP) IN) ExecuteOption[IN, DEP] {
	return func(c *ExecuteConfig[IN, DEP]) {
		c.retryConfig.Enabled = true
		c.retryConfig.InputHook = inputHookFunc
	}
}

// WithRetryConfig is an ExecuteOption that sets the retry configuration. This provides a way to
// customize the retry behavior specific to the needs of the operation. Use this for the most
// flexibility and control over the retry behavior.
func WithRetryConfig[IN, DEP any](config RetryConfig[IN, DEP]) ExecuteOption[IN, DEP] {
	return func(c *ExecuteConfig[IN, DEP]) {
		c.retryConfig = config
	}
}

// ExecuteOperation executes an operation with the given input and dependencies.
// Execution will return the previous successful execution result and skip execution if there was a
// previous successful run found in the Reports.
// If previous unsuccessful execution was found, the execution will not be skipped.
//
// Note:
// Operations that were skipped will not be added to the reporter.
//
// Retry:
// By default, it retries the operation up to 10 times with exponential backoff if it fails.
// Use WithRetryConfig to customize the retry behavior.
// To cancel the retry early, return an error with NewUnrecoverableError.
//
// Input & Output:
// The input and output must be JSON serializable. If the input is not serializable, it will return an error.
// To be serializable, the input and output must be json.marshalable, or it must implement json.Marshaler and json.Unmarshaler.
// IsSerializable can be used to check if the input or output is serializable.
func ExecuteOperation[IN, OUT, DEP any](
	b Bundle,
	operation *Operation[IN, OUT, DEP],
	deps DEP,
	input IN,
	opts ...ExecuteOption[IN, DEP],
) (Report[IN, OUT], error) {
	if !IsSerializable(b.Logger, input) {
		return Report[IN, OUT]{}, fmt.Errorf("operation %s input: %w", operation.def.ID, ErrNotSerializable)
	}

	if previousReport, found := loadPreviousSuccessfulReport[IN, OUT](b, operation.def, input); found {
		b.Logger.Infow("Operation already executed. Returning previous result", "id", operation.def.ID,
			"version", operation.def.Version, "description", operation.def.Description)

		return previousReport, nil
	}

	executeConfig := &ExecuteConfig[IN, DEP]{
		retryConfig: newDisabledRetryConfig[IN, DEP](),
	}
	for _, opt := range opts {
		opt(executeConfig)
	}

	var output OUT
	var err error

	if executeConfig.retryConfig.Enabled {
		var inputTemp = input

		// Generate the configurable options for the retry
		retryOpts := executeConfig.retryConfig.Policy.options()
		// Use the operation context in the retry
		retryOpts = append(retryOpts, retry.Context(b.GetContext()))
		// Append the retry logic which will log the retry and attempt to transform the input
		// if the user provided a custom input hook.
		retryOpts = append(retryOpts, retry.OnRetry(func(attempt uint, err error) {
			b.Logger.Infow("Operation failed. Retrying...",
				"operation", operation.def.ID, "attempt", attempt, "error", err)

			if executeConfig.retryConfig.InputHook != nil {
				inputTemp = executeConfig.retryConfig.InputHook(attempt, err, inputTemp, deps)
			}
		}))

		output, err = retry.DoWithData(
			func() (OUT, error) {
				return operation.execute(b, deps, inputTemp)
			},
			retryOpts...,
		)
	} else {
		output, err = operation.execute(b, deps, input)
	}

	if err == nil && !IsSerializable(b.Logger, output) {
		return Report[IN, OUT]{}, fmt.Errorf("operation %s output: %w", operation.def.ID, ErrNotSerializable)
	}

	report := NewReport(operation.def, input, output, err)
	if err = b.reporter.AddReport(genericReport(report)); err != nil {
		return Report[IN, OUT]{}, err
	}

	if report.Err != nil {
		return report, report.Err
	}

	return report, nil
}

// ExecuteSequence executes a Sequence and returns a SequenceReport.
// The SequenceReport contains a report for the Sequence and also the execution reports which are all
// the operations that were executed as part of this sequence.
// The latter is useful when we want to return all the executed reports to the changeset output.
// Execution will return the previous successful execution result and skip execution if there was a
// previous successful run found in the Reports.
// If previous unsuccessful execution was found, the execution will not be skipped.
//
// Note:
// Sequences or Operations that were skipped will not be added to the reporter.
// The ExecutionReports do not include Sequences or Operations that were skipped.
//
// Input & Output:
// The input and output must be JSON serializable. If the input is not serializable, it will return an error.
// To be serializable, the input and output must be json.marshalable, or it must implement json.Marshaler and json.Unmarshaler.
// IsSerializable can be used to check if the input or output is serializable.
func ExecuteSequence[IN, OUT, DEP any](
	b Bundle, sequence *Sequence[IN, OUT, DEP], deps DEP, input IN,
) (SequenceReport[IN, OUT], error) {
	if !IsSerializable(b.Logger, input) {
		return SequenceReport[IN, OUT]{}, fmt.Errorf("sequence %s input: %w", sequence.def.ID, ErrNotSerializable)
	}

	if previousReport, found := loadPreviousSuccessfulReport[IN, OUT](b, sequence.def, input); found {
		executionReports, err := b.reporter.GetExecutionReports(previousReport.ID)
		if err != nil {
			return SequenceReport[IN, OUT]{}, err
		}
		b.Logger.Infow("Sequence already executed. Returning previous result", "id", sequence.def.ID,
			"version", sequence.def.Version, "description", sequence.def.Description)

		return SequenceReport[IN, OUT]{previousReport, executionReports}, nil
	}

	b.Logger.Infow("Executing sequence", "id", sequence.def.ID,
		"version", sequence.def.Version, "description", sequence.def.Description)
	recentReporter := NewRecentMemoryReporter(b.reporter)
	newBundle := Bundle{
		Logger:          b.Logger,
		GetContext:      b.GetContext,
		reporter:        recentReporter,
		reportHashCache: b.reportHashCache,
	}
	ret, err := sequence.handler(newBundle, deps, input)
	if errors.Is(err, ErrNotSerializable) {
		return SequenceReport[IN, OUT]{}, err
	}

	if err == nil && !IsSerializable(b.Logger, ret) {
		return SequenceReport[IN, OUT]{}, fmt.Errorf("sequence %s output: %w", sequence.def.ID, ErrNotSerializable)
	}

	recentReports := recentReporter.GetRecentReports()
	childReports := make([]string, 0, len(recentReports))
	for _, rep := range recentReports {
		childReports = append(childReports, rep.ID)
	}

	report := NewReport(
		sequence.def,
		input,
		ret,
		err,
		childReports...,
	)

	if err = b.reporter.AddReport(genericReport(report)); err != nil {
		return SequenceReport[IN, OUT]{}, err
	}

	executionReports, err := b.reporter.GetExecutionReports(report.ID)
	if err != nil {
		return SequenceReport[IN, OUT]{}, err
	}

	if report.Err != nil {
		return SequenceReport[IN, OUT]{report, executionReports}, report.Err
	}

	return SequenceReport[IN, OUT]{report, executionReports}, nil
}

// NewUnrecoverableError creates an error that indicates an unrecoverable error.
// If this error is returned inside an operation, the operation will no longer retry.
// This allows the operation to fail fast if it encounters an unrecoverable error.
func NewUnrecoverableError(err error) error {
	return retry.Unrecoverable(err)
}

func loadPreviousSuccessfulReport[IN, OUT any](
	b Bundle, def Definition, input IN,
) (Report[IN, OUT], bool) {
	prevReports, err := b.reporter.GetReports()
	if err != nil {
		b.Logger.Errorw("Failed to get reports", "error", err)
		return Report[IN, OUT]{}, false
	}
	currentHash, err := constructUniqueHashFrom(b.reportHashCache, def, input)
	if err != nil {
		b.Logger.Errorw("Failed to construct unique hash", "error", err)
		return Report[IN, OUT]{}, false
	}

	for _, report := range prevReports {
		// Check if operation/sequence was run previously and return the report if successful
		reportHash, err := constructUniqueHashFrom(b.reportHashCache, report.Def, report.Input)
		if err != nil {
			b.Logger.Errorw("Failed to construct unique hash for previous report", "error", err)
			continue
		}
		if reportHash == currentHash && report.Err == nil {
			typedReport, ok := typeReport[IN, OUT](report)
			if !ok {
				b.Logger.Debugw(fmt.Sprintf("Previous %s execution found but couldn't find its matching Report", def.ID), "report_id", report.ID)
				continue
			}
			b.Logger.Debugw(fmt.Sprintf("Previous %s execution found. Returning its result from Report storage", def.ID), "report_id", report.ID)

			return typedReport, true
		}
	}

	// No previous execution was found
	return Report[IN, OUT]{}, false
}
