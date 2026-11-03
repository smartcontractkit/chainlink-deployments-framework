package mcms

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
)

type forkTestStatus string

const (
	forkTestStatusError   = forkTestStatus("error")
	forkTestStatusSuccess = forkTestStatus("success")
	forkTestStatusFailure = forkTestStatus("failure")
	forkTestStatusSkipped = forkTestStatus("skipped")
)

type ErrForkTestSkipped struct {
	message string
}

func (e ErrForkTestSkipped) Error() string {
	return e.message
}

type ErrForkTestFailed struct {
	cause error
	stage string
}

func (e ErrForkTestFailed) Error() string {
	return e.cause.Error()
}

type forkTestReport struct {
	ChainSelector uint64         `json:"chainSelector"`
	Status        forkTestStatus `json:"status"`
	Step          string         `json:"step,omitempty"`
	Details       string         `json:"details,omitempty"`
}

func newForkTestReport(chainSelector uint64, err error) forkTestReport {
	if err == nil {
		return forkTestReport{ChainSelector: chainSelector, Status: forkTestStatusSuccess}
	}

	if ferr, ok := errors.AsType[ErrForkTestSkipped](err); ok {
		return forkTestReport{
			ChainSelector: chainSelector,
			Status:        forkTestStatusSkipped,
			Details:       ferr.Error(),
		}
	}

	if ferr, ok := errors.AsType[ErrForkTestFailed](err); ok {
		return forkTestReport{
			ChainSelector: chainSelector,
			Status:        forkTestStatusFailure,
			Step:          ferr.stage,
			Details:       ferr.Error(),
		}
	}

	return forkTestReport{
		ChainSelector: chainSelector,
		Status:        forkTestStatusError,
		Details:       err.Error(),
	}
}

func (report forkTestReport) save(reportPath string) error {
	file, err := os.Create(reportPath)
	if err != nil {
		return fmt.Errorf("failed to create report file: %w", err)
	}
	defer file.Close()

	encoder := json.NewEncoder(file)
	encoder.SetIndent("", "  ")
	err = encoder.Encode(report)
	if err != nil {
		return fmt.Errorf("failed to encode report to JSON: %w", err)
	}

	return nil
}
