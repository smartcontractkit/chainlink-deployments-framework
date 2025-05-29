package operations

import (
	"encoding/json"
	"errors"
	"fmt"
	"sync"
	"time"

	"github.com/google/uuid"
)

// Report is the result of an operation.
// It contains the inputs and other metadata that was used to execute the operation.
type Report[IN, OUT any] struct {
	ID        string       `json:"id"`
	Def       Definition   `json:"definition"`
	Output    OUT          `json:"output"`
	Input     IN           `json:"input"`
	Timestamp *time.Time   `json:"timestamp"`
	Err       *ReportError `json:"error"`
	// stores a list of report ID for an operation that was executed as part of a sequence.
	ChildOperationReports []string `json:"childOperationReports"`
	// indicates if the operation was forced to run even if the same op was run previously with the same input.
	Forced bool `json:"forced,omitempty"`
}

// ToGenericReport converts the Report to a generic Report.
// This is useful when we want to return the report as a generic type in the changeset.output.
func (r Report[IN, OUT]) ToGenericReport() Report[any, any] {
	return genericReport(r)
}

// SequenceReport is a report for a sequence.
// It contains a report for the sequence itself and also a list of reports
// for all the operations executed as part of the sequence.
// The latter is useful when we want to return all the reports of the operations
// executed as part of the sequence in changeset output.
type SequenceReport[IN, OUT any] struct {
	Report[IN, OUT]

	// ExecutionReports is a list of report all the operations & sequence that was executed as part of this sequence.
	ExecutionReports []Report[any, any]
}

// ToGenericSequenceReport converts the SequenceReport to a generic SequenceReport.
// This is useful when we want to return the report as a generic type in the changeset.output.
func (r SequenceReport[IN, OUT]) ToGenericSequenceReport() SequenceReport[any, any] {
	return SequenceReport[any, any]{
		Report:           genericReport(r.Report),
		ExecutionReports: r.ExecutionReports,
	}
}

// NewReport creates a new report.
// ChildOperationReports is applicable only for Sequence.
func NewReport[IN, OUT any](
	def Definition, input IN, output OUT, err error, childReportsID ...string,
) Report[IN, OUT] {
	now := time.Now()
	r := Report[IN, OUT]{
		ID:                    uuid.New().String(),
		Def:                   def,
		Output:                output,
		Input:                 input,
		Timestamp:             &now,
		ChildOperationReports: childReportsID,
	}
	if err != nil {
		r.Err = &ReportError{Message: err.Error()}
	}

	return r
}

// ReportError represents an error in the Report.
// Its purpose is to have an exported field `Message` for marshalling as the
// native error cant be marshaled to JSON.
type ReportError struct {
	Message string `json:"message"`
}

// Error implements the error interface.
func (o ReportError) Error() string {
	return o.Message
}

var ErrReportNotFound = errors.New("report not found")

// Reporter manages reports. It can store them in memory, in the FS, etc.
type Reporter interface {
	GetReport(id string) (Report[any, any], error)
	GetReports() ([]Report[any, any], error)
	AddReport(report Report[any, any]) error
	GetExecutionReports(reportID string) ([]Report[any, any], error)
}

// MemoryReporter stores reports in memory.
// This is thread-safe and can be used in a multi-threaded environment.
type MemoryReporter struct {
	reports []Report[any, any]
	mu      sync.RWMutex
}

type MemoryReporterOption func(*MemoryReporter)

// WithReports is an option to initialize the MemoryReporter with a list of reports.
func WithReports(reports []Report[any, any]) MemoryReporterOption {
	return func(mr *MemoryReporter) {
		mr.reports = reports
	}
}

// NewMemoryReporter creates a new MemoryReporter.
// It can be initialized with a list of reports using the WithReports option.
func NewMemoryReporter(options ...MemoryReporterOption) *MemoryReporter {
	reporter := &MemoryReporter{}
	for _, opt := range options {
		opt(reporter)
	}

	return reporter
}

// AddReport adds a report to the memory reporter.
func (e *MemoryReporter) AddReport(report Report[any, any]) error {
	e.mu.Lock()
	defer e.mu.Unlock()

	e.reports = append(e.reports, report)

	return nil
}

// GetReports returns all reports.
func (e *MemoryReporter) GetReports() ([]Report[any, any], error) {
	e.mu.RLock()
	defer e.mu.RUnlock()

	// Create a copy to avoid data races after returning
	reports := make([]Report[any, any], len(e.reports))
	copy(reports, e.reports)

	return reports, nil
}

// GetReport returns a report by ID.
// Returns ErrReportNotFound if the report is not found.
func (e *MemoryReporter) GetReport(id string) (Report[any, any], error) {
	e.mu.RLock()
	defer e.mu.RUnlock()

	for _, report := range e.reports {
		if report.ID == id {
			return report, nil
		}
	}

	return Report[any, any]{}, fmt.Errorf("report_id %s: %w", id, ErrReportNotFound)
}

// GetExecutionReports returns all the reports that was executed as part of a sequence including itself.
// It does this by recursively fetching all the child reports.
// Useful when returning all the reports in a sequence to the changeset output.
func (e *MemoryReporter) GetExecutionReports(seqID string) ([]Report[any, any], error) {
	e.mu.RLock()
	defer e.mu.RUnlock()

	var allReports []Report[any, any]

	var getReportsRecursively func(id string) error
	getReportsRecursively = func(id string) error {
		var report Report[any, any]
		found := false

		for _, r := range e.reports {
			if r.ID == id {
				report = r
				found = true

				break
			}
		}

		if !found {
			return fmt.Errorf("report_id %s: %w", id, ErrReportNotFound)
		}

		for _, childID := range report.ChildOperationReports {
			if err := getReportsRecursively(childID); err != nil {
				return err
			}
		}
		allReports = append(allReports, report)

		return nil
	}

	if err := getReportsRecursively(seqID); err != nil {
		return nil, err
	}

	return allReports, nil
}

// RecentReporter is a wrapper around a Reporter that keeps track of the most recent reports.
// Useful when trying to get a list of reports that was recently added in a sequence.
// It is thread-safe and can be used in a multi-threaded environment.
type RecentReporter struct {
	Reporter
	recentReports []Report[any, any]
	mu            sync.RWMutex
}

// AddReport adds a report to the recent reporter.
func (e *RecentReporter) AddReport(report Report[any, any]) error {
	// First add to underlying reporter
	err := e.Reporter.AddReport(report)
	if err != nil {
		return err
	}

	// Then add to recent reports
	e.mu.Lock()
	defer e.mu.Unlock()

	e.recentReports = append(e.recentReports, report)

	return nil
}

// GetRecentReports returns all the reports that was added since the construction of the RecentReporter.
func (e *RecentReporter) GetRecentReports() []Report[any, any] {
	e.mu.RLock()
	defer e.mu.RUnlock()

	return e.recentReports
}

// NewRecentMemoryReporter creates a new RecentReporter.
func NewRecentMemoryReporter(reporter Reporter) *RecentReporter {
	r := &RecentReporter{
		Reporter:      reporter,
		recentReports: []Report[any, any]{},
	}

	return r
}

func genericReport[IN, OUT any](r Report[IN, OUT]) Report[any, any] {
	return Report[any, any]{
		ID: r.ID,
		Def: Definition{
			ID:          r.Def.ID,
			Version:     r.Def.Version,
			Description: r.Def.Description,
		},
		Output:                r.Output,
		Input:                 r.Input,
		Timestamp:             r.Timestamp,
		Err:                   r.Err,
		ChildOperationReports: r.ChildOperationReports,
		Forced:                r.Forced,
	}
}

// typeReport attempts to convert Report[any,any] type into Report[IN,OUT].
// This is needed when loading Report from disk and need to convert the type during execution
// once the type is known.
func typeReport[IN, OUT any](r Report[any, any]) (Report[IN, OUT], bool) {
	// When marshalling and unmarshalling, the type information is lost.
	// eg int becomes float64, struct becomes map[string]interface{}. So we need to unmarshal it
	// back to the original type as specified by the generic type to avoid data lost.
	inputBytes, err := json.Marshal(r.Input)
	if err != nil {
		return Report[IN, OUT]{}, false
	}
	var input IN
	if err = json.Unmarshal(inputBytes, &input); err != nil {
		return Report[IN, OUT]{}, false
	}

	outputBytes, err := json.Marshal(r.Output)
	if err != nil {
		return Report[IN, OUT]{}, false
	}

	var output OUT
	if err := json.Unmarshal(outputBytes, &output); err != nil {
		return Report[IN, OUT]{}, false
	}

	return Report[IN, OUT]{
		ID:                    r.ID,
		Def:                   r.Def,
		Output:                output,
		Input:                 input,
		Timestamp:             r.Timestamp,
		Err:                   r.Err,
		ChildOperationReports: r.ChildOperationReports,
		Forced:                r.Forced,
	}, true
}
