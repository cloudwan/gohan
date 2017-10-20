// Copyright (C) 2017 NTT Innovation Institute, Inc.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//    http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or
// implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package runner

import (
	"fmt"
	"os"
	"time"

	"github.com/mohae/deepcopy"
	"github.com/olekukonko/tablewriter"
	"github.com/onsi/ginkgo/config"
	"github.com/onsi/ginkgo/reporters/stenographer"
	"github.com/onsi/ginkgo/types"
)

const defaultStyle = "\x1b[0m"
const boldStyle = "\x1b[1m"
const redColor = "\x1b[91m"
const greenColor = "\x1b[32m"
const yellowColor = "\x1b[33m"
const cyanColor = "\x1b[36m"
const grayColor = "\x1b[90m"
const lightGrayColor = "\x1b[37m"

// Reporter is a custom test reporter which handles results from multiple test suites
type Reporter struct {
	suites []types.SuiteSummary
	specs  []types.SpecSummary
}

const (
	configSuccinct          = false
	configFullTrace         = true
	configNoisyPendings     = false
	configSlowSpecThreshold = float64(0.5) // sec
)

// AllSuitesSucceed returns whether tests within all suites passed
func (reporter *Reporter) AllSuitesSucceed() bool {
	for _, suite := range reporter.suites {
		if suite.NumberOfFailedSpecs != 0 {
			return false
		}
	}
	return true
}

// SpecSuiteWillBegin informs that suite will begin
func (reporter *Reporter) SpecSuiteWillBegin(config config.GinkgoConfigType, summary *types.SuiteSummary) {
}

// BeforeSuiteDidRun informs that before suite did run
func (reporter *Reporter) BeforeSuiteDidRun(setupSummary *types.SetupSummary) {
}

// SpecWillRun informs that spec will run
func (reporter *Reporter) SpecWillRun(specSummary *types.SpecSummary) {
}

// SpecDidComplete informs that spec did complete
func (reporter *Reporter) SpecDidComplete(specSummary *types.SpecSummary) {
	reporter.specs = append(reporter.specs, deepcopy.Copy(*specSummary).(types.SpecSummary))
}

// AfterSuiteDidRun informs that after suite did run
func (reporter *Reporter) AfterSuiteDidRun(setupSummary *types.SetupSummary) {
}

// SpecSuiteDidEnd informs that spec suite sis end
func (reporter *Reporter) SpecSuiteDidEnd(summary *types.SuiteSummary) {
	for i := range reporter.suites {
		if reporter.suites[i].SuiteDescription == summary.SuiteDescription {
			reporter.suites[i] = deepcopy.Copy(*summary).(types.SuiteSummary)
			break
		}
	}
}

// Prepare prepares a next test suite to run and zeroes its results
func (reporter *Reporter) Prepare(description string) {
	reporter.suites = append(reporter.suites, types.SuiteSummary{
		SuiteDescription: description,
		SuiteSucceeded:   true,
		SuiteID:          "undefined",
		NumberOfSpecsBeforeParallelization: 0,
		NumberOfTotalSpecs:                 0,
		NumberOfSpecsThatWillBeRun:         0,
		NumberOfPendingSpecs:               0,
		NumberOfSkippedSpecs:               0,
		NumberOfPassedSpecs:                0,
		NumberOfFailedSpecs:                0,
		RunTime:                            time.Duration(0),
	})
}

// Report prints the final report
func (reporter *Reporter) Report() {
	fmt.Println("--------------------------------------------------------------------------------")

	fmt.Println("Failures:")
	fmt.Println()

	steno := stenographer.New(true)

	for _, spec := range reporter.specs {
		steno.AnnounceCapturedOutput(spec.CapturedOutput)

		switch spec.State {
		case types.SpecStatePassed:
			if spec.IsMeasurement {
				steno.AnnounceSuccesfulMeasurement(&spec, configSuccinct)
			} else if spec.RunTime.Seconds() >= configSlowSpecThreshold {
				steno.AnnounceSuccesfulSlowSpec(&spec, configSuccinct)
			} else {
				steno.AnnounceSuccesfulSpec(&spec)
			}

		case types.SpecStatePending:
			steno.AnnouncePendingSpec(&spec, configNoisyPendings && !configSuccinct)
		case types.SpecStateSkipped:
			steno.AnnounceSkippedSpec(&spec, configSuccinct, configFullTrace)
		case types.SpecStateTimedOut:
			steno.AnnounceSpecTimedOut(&spec, configSuccinct, configFullTrace)
		case types.SpecStatePanicked:
			steno.AnnounceSpecPanicked(&spec, configSuccinct, configFullTrace)
		case types.SpecStateFailed:
			steno.AnnounceSpecFailed(&spec, configSuccinct, configFullTrace)
		}
	}

	fmt.Println()
	fmt.Println("Report:")
	fmt.Println()

	// total
	totalNumberOfTotalSpecs := 0
	totalNumberOfPassedSpecs := 0
	totalNumberOfFailedSpecs := 0
	totalNumberOfSkippedSpecs := 0
	totalNumberOfPendingSpecs := 0
	totalRunTime := time.Duration(0) * time.Nanosecond

	// prepare data
	data := [][]string{}

	// suites
	for index, suite := range reporter.suites {
		row := []string{}
		row = append(row, fmt.Sprintf("%d", index+1))

		if suite.NumberOfFailedSpecs == 0 {
			row = append(row, greenColor+suite.SuiteDescription+defaultStyle)
		} else {
			row = append(row, redColor+suite.SuiteDescription+defaultStyle)
		}

		row = append(row, cyanColor+fmt.Sprintf("%d", suite.NumberOfTotalSpecs)+defaultStyle)

		if suite.NumberOfFailedSpecs == 0 {
			row = append(row, greenColor+fmt.Sprintf("%d", suite.NumberOfPassedSpecs)+defaultStyle)
		} else {
			row = append(row, fmt.Sprintf("%d", suite.NumberOfPassedSpecs))
		}

		if suite.NumberOfFailedSpecs > 0 {
			row = append(row, redColor+fmt.Sprintf("%d", suite.NumberOfFailedSpecs)+defaultStyle)
		} else {
			row = append(row, fmt.Sprintf("%d", suite.NumberOfFailedSpecs))
		}

		if suite.NumberOfSkippedSpecs > 0 {
			row = append(row, yellowColor+fmt.Sprintf("%d", suite.NumberOfSkippedSpecs)+defaultStyle)
		} else {
			row = append(row, fmt.Sprintf("%d", suite.NumberOfSkippedSpecs))
		}

		if suite.NumberOfPendingSpecs > 0 {
			row = append(row, yellowColor+fmt.Sprintf("%d", suite.NumberOfPendingSpecs)+defaultStyle)
		} else {
			row = append(row, fmt.Sprintf("%d", suite.NumberOfPendingSpecs))
		}

		if suite.RunTime >= time.Duration(configSlowSpecThreshold*1000)*time.Millisecond {
			row = append(row, redColor+fmt.Sprintf("%.2f", float64(suite.RunTime.Nanoseconds())/1000000)+defaultStyle)
		} else {
			row = append(row, fmt.Sprintf("%.2f", float64(suite.RunTime.Nanoseconds())/1000000))
		}

		data = append(data, row)

		totalNumberOfTotalSpecs += suite.NumberOfTotalSpecs
		totalNumberOfPassedSpecs += suite.NumberOfPassedSpecs
		totalNumberOfFailedSpecs += suite.NumberOfFailedSpecs
		totalNumberOfSkippedSpecs += suite.NumberOfSkippedSpecs
		totalNumberOfPendingSpecs += suite.NumberOfPendingSpecs
		totalRunTime += suite.RunTime
	}

	footer := []string{"", boldStyle + cyanColor + "SUMMARY" + defaultStyle}

	footer = append(footer, boldStyle+cyanColor+fmt.Sprintf("%d", totalNumberOfTotalSpecs)+defaultStyle)

	if totalNumberOfFailedSpecs == 0 {
		footer = append(footer, boldStyle+greenColor+fmt.Sprintf("%d", totalNumberOfPassedSpecs)+defaultStyle)
	} else {
		footer = append(footer, boldStyle+fmt.Sprintf("%d", totalNumberOfPassedSpecs)+defaultStyle)
	}
	if totalNumberOfFailedSpecs > 0 {
		footer = append(footer, boldStyle+redColor+fmt.Sprintf("%d", totalNumberOfFailedSpecs)+defaultStyle)
	} else {
		footer = append(footer, boldStyle+fmt.Sprintf("%d", totalNumberOfFailedSpecs)+defaultStyle)
	}
	if totalNumberOfSkippedSpecs > 0 {
		footer = append(footer, boldStyle+yellowColor+fmt.Sprintf("%d", totalNumberOfSkippedSpecs)+defaultStyle)
	} else {
		footer = append(footer, boldStyle+fmt.Sprintf("%d", totalNumberOfSkippedSpecs)+defaultStyle)
	}
	if totalNumberOfPendingSpecs > 0 {
		footer = append(footer, boldStyle+yellowColor+fmt.Sprintf("%d", totalNumberOfPendingSpecs)+defaultStyle)
	} else {
		footer = append(footer, boldStyle+fmt.Sprintf("%d", totalNumberOfPendingSpecs)+defaultStyle)
	}
	footer = append(footer, boldStyle+fmt.Sprintf("%.2f", float64(totalRunTime.Nanoseconds())/1000000)+defaultStyle)

	data = append(data, footer)

	table := tablewriter.NewWriter(os.Stdout)
	table.SetHeader([]string{"No.", "Name", "Total", "Passed", "Failed", "Skipped", "Pending", "Time [ms]"})
	table.SetAlignment(tablewriter.ALIGN_RIGHT)
	table.AppendBulk(data)
	table.Render()
}

// NewReporter allocates a new reporter
func NewReporter() *Reporter {
	return &Reporter{
		suites: []types.SuiteSummary{},
	}
}
