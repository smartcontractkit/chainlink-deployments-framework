package operations

import (
	"context"
	"errors"
	"math"
	"sync"
	"testing"
	"time"

	"github.com/Masterminds/semver/v3"
	"github.com/smartcontractkit/chainlink-common/pkg/logger"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func Test_ExecuteOperation(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name              string
		options           []ExecuteOption[int, any]
		IsUnrecoverable   bool
		wantOpCalledTimes int
		wantOutput        int
		wantErr           string
	}{
		{
			name:              "no retry",
			wantOpCalledTimes: 1,
			wantErr:           "test error",
		},
		{
			name: "with default retry",
			options: []ExecuteOption[int, any]{
				WithRetry[int, any](),
			},
			wantOpCalledTimes: 3,
			wantOutput:        2,
		},
		{
			name: "with custom retry eventual success",
			options: []ExecuteOption[int, any]{
				WithRetryConfig(RetryConfig[int, any]{
					Enabled: true,
					Policy: RetryPolicy{
						MaxAttempts: 10,
					},
				}),
			},
			wantOpCalledTimes: 3,
			wantOutput:        2,
		},
		{
			name: "with custom retry eventual failure",
			options: []ExecuteOption[int, any]{
				WithRetryConfig(RetryConfig[int, any]{
					Enabled: true,
					Policy: RetryPolicy{
						MaxAttempts: 1,
					},
				}),
			},
			wantOpCalledTimes: 1,
			wantErr:           "test error",
		},
		{
			name: "NewInputHook",
			options: []ExecuteOption[int, any]{
				WithRetryInput(func(attempt uint, err error, input int, deps any) int {
					require.ErrorContains(t, err, "test error")
					// update input to 5 after first failed attempt
					return 5
				}),
			},
			wantOpCalledTimes: 3,
			wantOutput:        6,
		},
		{
			name:              "UnrecoverableError",
			IsUnrecoverable:   true,
			wantOpCalledTimes: 1,
			wantErr:           "fatal error",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			failTimes := 2
			handlerCalledTimes := 0
			handler := func(b Bundle, deps any, input int) (output int, err error) {
				handlerCalledTimes++
				if tt.IsUnrecoverable {
					return 0, NewUnrecoverableError(errors.New("fatal error"))
				}

				if failTimes > 0 {
					failTimes--
					return 0, errors.New("test error")
				}

				return input + 1, nil
			}
			op := NewOperation("plus1", semver.MustParse("1.0.0"), "test operation", handler)
			e := NewBundle(context.Background, logger.Test(t), NewMemoryReporter())

			res, err := ExecuteOperation(e, op, nil, 1, tt.options...)

			if tt.wantErr != "" {
				require.Error(t, res.Err)
				require.Error(t, err)
				require.ErrorContains(t, res.Err, tt.wantErr)
				require.ErrorContains(t, err, tt.wantErr)
			} else {
				require.Nil(t, res.Err)
				require.NoError(t, err)
				assert.Equal(t, tt.wantOutput, res.Output)
			}
			assert.Equal(t, tt.wantOpCalledTimes, handlerCalledTimes)
			// check report is added to reporter
			report, err := e.reporter.GetReport(res.ID)
			require.NoError(t, err)
			assert.NotNil(t, report)
		})
	}
}

func Test_ExecuteOperation_ErrorReporter(t *testing.T) {
	t.Parallel()

	op := NewOperation("plus1", semver.MustParse("1.0.0"), "test operation",
		func(e Bundle, deps any, input int) (output int, err error) {
			return input + 1, nil
		})

	reportErr := errors.New("add report error")
	errReporter := errorReporter{
		Reporter:       NewMemoryReporter(),
		AddReportError: reportErr,
	}
	e := NewBundle(context.Background, logger.Test(t), errReporter)

	res, err := ExecuteOperation(e, op, nil, 1)
	require.Error(t, err)
	require.ErrorContains(t, err, reportErr.Error())
	require.Nil(t, res.Err)
}

func Test_ExecuteOperation_WithPreviousRun(t *testing.T) {
	t.Parallel()

	handlerCalledTimes := 0
	handler := func(b Bundle, deps any, input int) (output int, err error) {
		handlerCalledTimes++
		return input + 1, nil
	}
	handlerWithErrorCalledTimes := 0
	handlerWithError := func(b Bundle, deps any, input int) (output int, err error) {
		handlerWithErrorCalledTimes++
		return 0, NewUnrecoverableError(errors.New("test error"))
	}

	op := NewOperation("plus1", semver.MustParse("1.0.0"), "test operation", handler)
	opWithError := NewOperation("plus1-error", semver.MustParse("1.0.0"), "test operation error", handlerWithError)
	bundle := NewBundle(t.Context, logger.Test(t), NewMemoryReporter())

	// first run
	res, err := ExecuteOperation(bundle, op, nil, 1)
	require.NoError(t, err)
	require.Nil(t, res.Err)
	assert.Equal(t, 2, res.Output)
	assert.Equal(t, 1, handlerCalledTimes)

	// rerun should return previous report
	res, err = ExecuteOperation(bundle, op, nil, 1)
	require.NoError(t, err)
	require.Nil(t, res.Err)
	assert.Equal(t, 2, res.Output)
	assert.Equal(t, 1, handlerCalledTimes)

	// new run with different input, should perform execution
	res, err = ExecuteOperation(bundle, op, nil, 3)
	require.NoError(t, err)
	require.Nil(t, res.Err)
	assert.Equal(t, 4, res.Output)
	assert.Equal(t, 2, handlerCalledTimes)

	// new run with different op, should perform execution
	op = NewOperation("plus1-v2", semver.MustParse("2.0.0"), "test operation", handler)
	res, err = ExecuteOperation(bundle, op, nil, 1)
	require.NoError(t, err)
	require.Nil(t, res.Err)
	assert.Equal(t, 2, res.Output)
	assert.Equal(t, 3, handlerCalledTimes)

	// new run with op that returns error
	res, err = ExecuteOperation(bundle, opWithError, nil, 1)
	require.Error(t, err)
	require.ErrorContains(t, err, "test error")
	require.ErrorContains(t, res.Err, "test error")
	assert.Equal(t, 1, handlerWithErrorCalledTimes)

	// rerun with op that returns error, should attempt execution again
	res, err = ExecuteOperation(bundle, opWithError, nil, 1)
	require.Error(t, err)
	require.ErrorContains(t, err, "test error")
	require.ErrorContains(t, res.Err, "test error")
	assert.Equal(t, 2, handlerWithErrorCalledTimes)
}

func Test_ExecuteOperation_Unserializable_Data(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name      string
		input     any
		output    any
		wantError string
	}{
		{
			name:   "both input and output are serializable",
			input:  1,
			output: 2,
		},
		{
			name:      "input is serializable, output is not",
			input:     1,
			output:    func() bool { return true },
			wantError: "operation example output: data cannot be safely written to disk without data lost, avoid type that can't be serialized",
		},
		{
			name: "input is not serializable, output is",
			input: struct {
				A            int
				privateField string
			}{
				A:            1,
				privateField: "private",
			},
			output:    2,
			wantError: "operation example input: data cannot be safely written to disk without data lost, avoid type that can't be serialized",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			op := NewOperation("example", semver.MustParse("1.0.0"), "test operation",
				func(e Bundle, deps any, input any) (output any, err error) {
					return tt.output, nil
				})

			e := NewBundle(context.Background, logger.Test(t), NewMemoryReporter())

			res, err := ExecuteOperation(e, op, nil, tt.input)
			if len(tt.wantError) != 0 {
				require.Error(t, err)
				require.ErrorContains(t, err, tt.wantError)
			} else {
				require.NoError(t, err)
				require.Nil(t, res.Err)
			}
		})
	}
}

func Test_ExecuteSequence(t *testing.T) {
	t.Parallel()

	version := semver.MustParse("1.0.0")

	tests := []struct {
		name            string
		simulateOpError bool
		wantOutput      int
		wantErr         string
	}{
		{
			name:       "Success Execution",
			wantOutput: 3,
		},
		{
			name:            "Error Execution",
			simulateOpError: true,
			wantErr:         "fatal error",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			op := NewOperation("plus1", version, "plus 1",
				func(e Bundle, deps OpDeps, input int) (output int, err error) {
					if tt.simulateOpError {
						return 0, NewUnrecoverableError(errors.New("fatal error"))
					}

					return input + 1, nil
				})

			var opID string
			sequence := NewSequence("seq-plus1", version, "plus 1",
				func(env Bundle, deps any, input int) (int, error) {
					res, err := ExecuteOperation(env, op, OpDeps{}, input)
					// capture for verification later
					opID = res.ID
					if err != nil {
						return 0, err
					}

					return res.Output + 1, nil
				})

			e := NewBundle(context.Background, logger.Test(t), NewMemoryReporter())

			seqReport, err := ExecuteSequence(e, sequence, nil, 1)

			if tt.simulateOpError {
				require.Error(t, seqReport.Err)
				require.Error(t, err)
				require.ErrorContains(t, seqReport.Err, tt.wantErr)
				require.ErrorContains(t, err, tt.wantErr)
			} else {
				require.Nil(t, seqReport.Err)
				require.NoError(t, err)
				assert.Equal(t, tt.wantOutput, seqReport.Output)
			}
			assert.Equal(t, []string{opID}, seqReport.ChildOperationReports)
			// check report is added to reporter
			report, err := e.reporter.GetReport(seqReport.ID)
			require.NoError(t, err)
			assert.NotNil(t, report)
			assert.Len(t, seqReport.ExecutionReports, 2) // 1 seq report + 1 op report

			// check allReports contain the parent and child reports
			childReport, err := e.reporter.GetReport(opID)
			require.NoError(t, err)
			assert.Equal(t, seqReport.ExecutionReports[0], childReport)
			assert.Equal(t, seqReport.ExecutionReports[1], report)
		})
	}
}

func Test_ExecuteSequence_WithPreviousRun(t *testing.T) {
	t.Parallel()

	version := semver.MustParse("1.0.0")
	op := NewOperation("plus1", version, "plus 1",
		func(b Bundle, deps OpDeps, input int) (output int, err error) {
			return input + 1, nil
		})

	handlerCalledTimes := 0
	handler := func(b Bundle, deps any, input int) (int, error) {
		handlerCalledTimes++
		res, err := ExecuteOperation(b, op, OpDeps{}, input)
		if err != nil {
			return 0, err
		}

		return res.Output, nil
	}
	handlerWithErrorCalledTimes := 0
	handlerWithError := func(b Bundle, deps any, input int) (int, error) {
		handlerWithErrorCalledTimes++
		return 0, NewUnrecoverableError(errors.New("test error"))
	}
	sequence := NewSequence("seq-plus1", version, "plus 1", handler)
	sequenceWithError := NewSequence("seq-plus1-error", version, "plus 1 error", handlerWithError)

	bundle := NewBundle(context.Background, logger.Test(t), NewMemoryReporter())

	// first run
	res, err := ExecuteSequence(bundle, sequence, nil, 1)
	require.NoError(t, err)
	require.Nil(t, res.Err)
	assert.Equal(t, 2, res.Output)
	assert.Len(t, res.ExecutionReports, 2) // 1 seq report + 1 op report
	assert.Equal(t, 1, handlerCalledTimes)

	// rerun should return previous report
	res, err = ExecuteSequence(bundle, sequence, nil, 1)
	require.NoError(t, err)
	require.Nil(t, res.Err)
	assert.Equal(t, 2, res.Output)
	assert.Len(t, res.ExecutionReports, 2) // 1 seq report + 1 op report
	assert.Equal(t, 1, handlerCalledTimes)

	// new run with different input, should perform execution
	res, err = ExecuteSequence(bundle, sequence, nil, 3)
	require.NoError(t, err)
	require.Nil(t, res.Err)
	assert.Equal(t, 4, res.Output)
	assert.Len(t, res.ExecutionReports, 2) // 1 seq report + 1 op report
	assert.Equal(t, 2, handlerCalledTimes)

	// new run with different sequence but same operation, should perform execution
	sequence = NewSequence("seq-plus1-v2", semver.MustParse("2.0.0"), "plus 1", handler)
	res, err = ExecuteSequence(bundle, sequence, nil, 1)
	require.NoError(t, err)
	require.Nil(t, res.Err)
	assert.Equal(t, 2, res.Output)
	// only 1 because the op was not executed due to previous execution found
	assert.Len(t, res.ExecutionReports, 1)
	assert.Equal(t, 3, handlerCalledTimes)

	// new run with sequence that returns error
	res, err = ExecuteSequence(bundle, sequenceWithError, nil, 1)
	require.Error(t, err)
	require.ErrorContains(t, err, "test error")
	require.ErrorContains(t, res.Err, "test error")
	assert.Equal(t, 1, handlerWithErrorCalledTimes)

	// rerun with sequence that returns error, should attempt execution again
	res, err = ExecuteSequence(bundle, sequenceWithError, nil, 1)
	require.Error(t, err)
	require.ErrorContains(t, err, "test error")
	require.ErrorContains(t, res.Err, "test error")
	assert.Equal(t, 2, handlerWithErrorCalledTimes)
}

func Test_ExecuteSequence_ErrorReporter(t *testing.T) {
	t.Parallel()

	version := semver.MustParse("1.0.0")
	op := NewOperation("plus1", version, "plus 1",
		func(e Bundle, deps OpDeps, input int) (output int, err error) {
			return input + 1, nil
		})

	sequence := NewSequence("seq-plus1", version, "plus 1",
		func(env Bundle, deps OpDeps, input int) (int, error) {
			res, err := ExecuteOperation(env, op, OpDeps{}, input)
			if err != nil {
				return 0, err
			}

			return res.Output + 1, nil
		})

	tests := []struct {
		name          string
		setupReporter func() Reporter
		wantErr       string
	}{
		{
			name: "AddReport returns an error",
			setupReporter: func() Reporter {
				return errorReporter{
					Reporter:       NewMemoryReporter(),
					AddReportError: errors.New("add report error"),
				}
			},
			wantErr: "add report error",
		},
		{
			name: "GetExecutionReports returns an error",
			setupReporter: func() Reporter {
				return errorReporter{
					Reporter:                 NewMemoryReporter(),
					GetExecutionReportsError: errors.New("get execution reports error"),
				}
			},
			wantErr: "get execution reports error",
		},
		{
			name: "Loaded previous report but GetExecutionReports returns an error",
			setupReporter: func() Reporter {
				r := errorReporter{
					Reporter:                 NewMemoryReporter(),
					GetExecutionReportsError: errors.New("get execution reports error"),
				}
				err := r.AddReport(genericReport(
					NewReport(sequence.def, 1, 2, nil),
				))
				require.NoError(t, err)

				return r
			},
			wantErr: "get execution reports error",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			e := NewBundle(context.Background, logger.Test(t), tt.setupReporter())
			_, err := ExecuteSequence(e, sequence, OpDeps{}, 1)
			require.Error(t, err)
			require.ErrorContains(t, err, tt.wantErr)
		})
	}
}

func Test_ExecuteSequence_Unserializable_Data(t *testing.T) {
	t.Parallel()

	version := semver.MustParse("1.0.0")
	op := NewOperation("test", version, "test description",
		func(b Bundle, deps OpDeps, input any) (output any, err error) {
			return 1, nil
		})

	tests := []struct {
		name      string
		input     any
		output    any
		wantError string
	}{
		{
			name:   "both input and output are serializable",
			input:  1,
			output: 2,
		},
		{
			name:      "input is serializable, output is not",
			input:     1,
			output:    func() bool { return true },
			wantError: "sequence seq-example output: data cannot be safely written to disk without data lost, avoid type that can't be serialized",
		},
		{
			name: "input is not serializable, output is",
			input: struct {
				A            int
				privateField string
			}{
				A:            1,
				privateField: "private",
			},
			output:    2,
			wantError: "sequence seq-example input: data cannot be safely written to disk without data lost, avoid type that can't be serialized",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			sequence := NewSequence("seq-example", version, "test operation",
				func(e Bundle, deps any, _ any) (output any, err error) {
					_, err = ExecuteOperation(e, op, OpDeps{}, 1)
					if err != nil {
						return 0, err
					}

					return tt.output, nil
				})

			e := NewBundle(context.Background, logger.Test(t), NewMemoryReporter())

			res, err := ExecuteSequence(e, sequence, nil, tt.input)
			if len(tt.wantError) != 0 {
				require.Error(t, err)
				require.ErrorContains(t, err, tt.wantError)
			} else {
				require.NoError(t, err)
				require.Nil(t, res.Err)
			}
		})
	}
}

func Test_loadPreviousSuccessfulReport(t *testing.T) {
	t.Parallel()

	version := semver.MustParse("1.0.0")
	definition := Definition{
		ID:          "plus1",
		Version:     version,
		Description: "plus 1",
	}

	tests := []struct {
		name          string
		setupReporter func() Reporter
		input         float64
		wantDef       Definition
		wantInput     float64
		wantFound     bool
	}{
		{
			name: "Failed to GetReports",
			setupReporter: func() Reporter {
				return errorReporter{
					GetReportsError: errors.New("failed to get reports"),
				}
			},
			input:     1,
			wantFound: false,
		},
		{
			name: "Successful Report found - return report",
			setupReporter: func() Reporter {
				r := NewMemoryReporter()
				err := r.AddReport(genericReport(
					NewReport(definition, 1, 2, nil),
				))
				require.NoError(t, err)

				return r
			},
			input:     1,
			wantDef:   definition,
			wantInput: 1,
			wantFound: true,
		},
		{
			name: "Report with error found - ignore report",
			setupReporter: func() Reporter {
				r := NewMemoryReporter()
				err := r.AddReport(genericReport(
					NewReport(definition, 1, 2, errors.New("failed")),
				))
				require.NoError(t, err)

				return r
			},
			input:     1,
			wantFound: false,
		},
		{
			name:      "Report not found",
			input:     1,
			wantFound: false,
		},
		{
			name:      "Current report with bad hash",
			input:     math.NaN(),
			wantFound: false,
		},
		{
			name: "Previous report with bad hash",
			setupReporter: func() Reporter {
				r := NewMemoryReporter()
				err := r.AddReport(genericReport(
					NewReport(definition, math.NaN(), 2, nil),
				))
				require.NoError(t, err)

				return r
			},
			input:     1,
			wantFound: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			bundle := NewBundle(context.Background, logger.Test(t), NewMemoryReporter())
			if tt.setupReporter != nil {
				bundle.reporter = tt.setupReporter()
			}

			report, found := loadPreviousSuccessfulReport[float64, int](bundle, definition, tt.input)
			assert.Equal(t, tt.wantFound, found)

			if tt.wantFound {
				assert.Equal(t, tt.wantDef, report.Def)
				assert.InDelta(t, tt.wantInput, report.Input, 0)
			}
		})
	}
}

func Test_ExecuteSequence_Concurrent(t *testing.T) {
	t.Parallel()

	version := semver.MustParse("1.0.0")

	op := NewOperation("increment", version, "increment by 1",
		func(b Bundle, deps any, input int) (output int, err error) {
			return input + 1, nil
		})

	sequence := NewSequence("concurrent-seq", version, "concurrent sequence test",
		func(b Bundle, deps any, input int) (int, error) {
			res, err := ExecuteOperation(b, op, nil, input)
			if err != nil {
				return 0, err
			}

			// Introduce a small delay to increase chance of race conditions
			time.Sleep(time.Millisecond)

			return res.Output, nil
		})

	reporter := NewMemoryReporter()
	bundle := NewBundle(context.Background, logger.Test(t), reporter)

	const numGoroutines = 10
	var wg sync.WaitGroup
	wg.Add(numGoroutines)

	// Channel to collect results
	type result struct {
		report SequenceReport[int, int]
		err    error
	}
	results := make(chan result, numGoroutines)

	for i := range numGoroutines {
		go func(input int) {
			defer wg.Done()

			report, err := ExecuteSequence(bundle, sequence, nil, input)
			results <- result{report, err}
		}(i) // Each goroutine uses its index as input
	}

	wg.Wait()
	close(results)

	// Collect and verify results
	for res := range results {
		require.NoError(t, res.err, "ExecuteSequence should not return an error")
		require.Nil(t, res.report.Err, "Report error should be nil")

		// Output should be input + 1
		input := res.report.Input
		expectedOutput := input + 1
		assert.Equal(t, expectedOutput, res.report.Output,
			"Output should be input + 1 for input %d", input)

		// Verify execution reports
		assert.Len(t, res.report.ExecutionReports, 2,
			"Should have 2 execution reports (sequence + operation)")
	}

	// Verify reporter has all reports
	allReports, err := reporter.GetReports()
	require.NoError(t, err)

	// We expect 2*numGoroutines reports (1 sequence + 1 operation per goroutine)
	assert.Len(t, allReports, numGoroutines*2,
		"Reporter should have %d reports", numGoroutines*2)
}

func Test_ExecuteOperation_Concurrent(t *testing.T) {
	t.Parallel()

	version := semver.MustParse("1.0.0")

	op := NewOperation("increment", version, "increment by 1",
		func(b Bundle, deps any, input int) (output int, err error) {
			// Introduce a small delay to increase chance of race conditions
			time.Sleep(time.Millisecond)
			return input + 1, nil
		})

	reporter := NewMemoryReporter()
	bundle := NewBundle(context.Background, logger.Test(t), reporter)

	const numGoroutines = 10
	var wg sync.WaitGroup
	wg.Add(numGoroutines)

	// Channel to collect results
	type result struct {
		report Report[int, int]
		err    error
	}
	results := make(chan result, numGoroutines)

	for i := range numGoroutines {
		go func(input int) {
			defer wg.Done()

			report, err := ExecuteOperation(bundle, op, nil, input)
			results <- result{report, err}
		}(i) // Each goroutine uses its index as input
	}

	wg.Wait()
	close(results)

	for res := range results {
		require.NoError(t, res.err, "ExecuteOperation should not return an error")
		require.Nil(t, res.report.Err, "Report error should be nil")

		// Output should be input + 1
		input := res.report.Input
		expectedOutput := input + 1
		assert.Equal(t, expectedOutput, res.report.Output,
			"Output should be input + 1 for input %d", input)
	}

	// Verify reporter has all reports
	allReports, err := reporter.GetReports()
	require.NoError(t, err)

	// We expect numGoroutines reports (1 per goroutine)
	assert.Len(t, allReports, numGoroutines,
		"Reporter should have %d reports", numGoroutines)
}

type errorReporter struct {
	Reporter
	GetReportError           error
	GetReportsError          error
	AddReportError           error
	GetExecutionReportsError error
}

func (e errorReporter) GetReport(id string) (Report[any, any], error) {
	if e.GetReportError != nil {
		return Report[any, any]{}, e.GetReportError
	}

	return e.Reporter.GetReport(id)
}

func (e errorReporter) GetReports() ([]Report[any, any], error) {
	if e.GetReportsError != nil {
		return nil, e.GetReportsError
	}

	return e.Reporter.GetReports()
}

func (e errorReporter) AddReport(report Report[any, any]) error {
	if e.AddReportError != nil {
		return e.AddReportError
	}

	return e.Reporter.AddReport(report)
}

func (e errorReporter) GetExecutionReports(id string) ([]Report[any, any], error) {
	if e.GetExecutionReportsError != nil {
		return nil, e.GetExecutionReportsError
	}

	return e.Reporter.GetExecutionReports(id)
}
