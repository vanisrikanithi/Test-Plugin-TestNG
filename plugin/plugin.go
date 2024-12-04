package plugin

import (
	"context"
	"encoding/xml"
	"errors"
	"os"
	"path/filepath"
	"strconv"

	"github.com/sirupsen/logrus"
)

// Args represents the plugin's configurable arguments.
type Args struct {
	ReportFilenamePattern     string `envconfig:"PLUGIN_REPORT_FILENAME_PATTERN"`
	FailedFails               int    `envconfig:"PLUGIN_FAILED_FAILS"`
	FailedSkips               int    `envconfig:"PLUGIN_FAILED_SKIPS"`
	FailureOnFailedTestConfig bool   `envconfig:"PLUGIN_FAILURE_ON_FAILED_TEST_CONFIG"`
	UnstableFails             int    `envconfig:"PLUGIN_UNSTABLE_FAILS"`
	UnstableSkips             int    `envconfig:"PLUGIN_UNSTABLE_SKIPS"`
	ThresholdMode             int    `envconfig:"PLUGIN_THRESHOLD_MODE"`
	PluginFailIfNoResults     bool   `envconfig:"PLUGIN_FAIL_IF_NO_RESULTS"`
	Level                     string `envconfig:"PLUGIN_LOG_LEVEL"`
}

// ValidateInputs ensures the user inputs meet the plugin requirements.
func ValidateInputs(args Args) error {
	if args.ReportFilenamePattern == "" {
		return errors.New("missing required parameter: ReportFilenamePattern")
	}
	if args.FailedFails < 0 || args.FailedSkips < 0 || args.UnstableFails < 0 || args.UnstableSkips < 0 {
		return errors.New("threshold values must be non-negative")
	}
	if args.ThresholdMode != 1 && args.ThresholdMode != 2 {
		return errors.New("thresholdMode must be 1 (absolute) or 2 (percentage)")
	}
	return nil
}

// Exec handles TestNG XML report processing and conversion to JUnit XML.
func Exec(ctx context.Context, args Args) error {
	files, err := locateFiles(args.ReportFilenamePattern)
	if err != nil {
		return err
	}

	if args.PluginFailIfNoResults && len(files) == 0 {
		return errors.New("no TestNG XML report files found")
	}

	for _, file := range files {
		if err := processFile(file, args); err != nil {
			return err
		}
	}

	return nil
}

// locateFiles identifies files matching the given pattern.
func locateFiles(pattern string) ([]string, error) {
	matches, err := filepath.Glob(pattern)
	if err != nil || len(matches) == 0 {
		return nil, errors.New("no files found matching the report filename pattern")
	}
	return matches, nil
}

// processFile reads a TestNG XML, validates thresholds, converts it to JUnit XML, and writes it back.
func processFile(filename string, args Args) error {
	logrus.Infof("Processing file: %s", filename)

	data, err := os.ReadFile(filename)
	if err != nil {
		return err
	}

	var testNGReport TestNGReport
	if err := xml.Unmarshal(data, &testNGReport); err != nil {
		return errors.New("failed to parse TestNG XML: " + err.Error())
	}

	if err := validateThresholds(testNGReport, args); err != nil {
		return err
	}

	junitReport, err := convertToJUnit(testNGReport)
	if err != nil {
		return err
	}

	outputData, err := xml.MarshalIndent(junitReport, "", "  ")
	if err != nil {
		return errors.New("failed to marshal JUnit XML: " + err.Error())
	}

	if err := os.WriteFile(filename, outputData, 0644); err != nil {
		return err
	}

	logrus.Infof("Successfully converted %s to JUnit format", filename)
	return nil
}

// validateThresholds checks thresholds like failed/skipped tests and configuration failures.
func validateThresholds(report TestNGReport, args Args) error {
	for _, suite := range report.Suites {
		// Check thresholds based on the mode (absolute or percentage)
		if args.ThresholdMode == 1 {
			if err := validateAbsoluteThresholds(suite, args); err != nil {
				return err
			}
		} else if args.ThresholdMode == 2 {
			if err := validatePercentageThresholds(suite, args); err != nil {
				return err
			}
		} else {
			return errors.New("invalid thresholdMode: must be 1 (absolute) or 2 (percentage)")
		}

		// Check for unstable thresholds
		checkUnstableThresholds(suite, args)

		// Check for configuration failures
		if err := validateConfigFailures(suite, args); err != nil {
			return err
		}
	}
	return nil
}

// validateAbsoluteThresholds checks absolute thresholds for failures and skips.
func validateAbsoluteThresholds(suite Suite, args Args) error {
	if args.FailedFails > 0 && suite.Failures > args.FailedFails {
		return errors.New(
			"number of failed tests exceeded the failure threshold: " +
				"provided threshold=" + strconv.Itoa(args.FailedFails) +
				", actual failed=" + strconv.Itoa(suite.Failures),
		)
	}
	if args.FailedSkips > 0 && suite.Skipped > args.FailedSkips {
		return errors.New(
			"number of skipped tests exceeded the failure threshold: " +
				"provided threshold=" + strconv.Itoa(args.FailedSkips) +
				", actual skipped=" + strconv.Itoa(suite.Skipped),
		)
	}
	return nil
}

// validatePercentageThresholds checks percentage-based thresholds for failures and skips.
func validatePercentageThresholds(suite Suite, args Args) error {
	totalTests := suite.Tests
	if totalTests == 0 {
		return nil // Avoid division by zero
	}

	failureRate := float64(suite.Failures) / float64(totalTests) * 100
	skipRate := float64(suite.Skipped) / float64(totalTests) * 100

	if args.FailedFails > 0 && failureRate > float64(args.FailedFails) {
		return errors.New(
			"failure rate exceeded the failure threshold: " +
				"provided threshold=" + strconv.Itoa(args.FailedFails) +
				"%, actual failure rate=" + strconv.FormatFloat(failureRate, 'f', 2, 64) + "%",
		)
	}
	if args.FailedSkips > 0 && skipRate > float64(args.FailedSkips) {
		return errors.New(
			"skip rate exceeded the failure threshold: " +
				"provided threshold=" + strconv.Itoa(args.FailedSkips) +
				"%, actual skip rate=" + strconv.FormatFloat(skipRate, 'f', 2, 64) + "%",
		)
	}
	return nil
}

// checkUnstableThresholds logs warnings for unstable thresholds for failures and skips.
func checkUnstableThresholds(suite Suite, args Args) {
	if args.UnstableFails > 0 && suite.Failures > args.UnstableFails {
		logrus.Warnf(
			"Number of failed tests exceeded unstable threshold: "+
				"provided threshold=%d, actual failed=%d; marking build as UNSTABLE",
			args.UnstableFails, suite.Failures,
		)
	}
	if args.UnstableSkips > 0 && suite.Skipped > args.UnstableSkips {
		logrus.Warnf(
			"Number of skipped tests exceeded unstable threshold: "+
				"provided threshold=%d, actual skipped=%d; marking build as UNSTABLE",
			args.UnstableSkips, suite.Skipped,
		)
	}
}

// validateConfigFailures checks for failed configuration methods and returns an error if any exist.
func validateConfigFailures(suite Suite, args Args) error {
	if args.FailureOnFailedTestConfig {
		for _, class := range suite.Classes {
			for _, test := range class.Tests {
				if test.IsConfig && test.Status == "FAIL" {
					return errors.New(
						"a configuration method failed: class=" + class.Name +
							", method=" + test.Name,
					)
				}
			}
		}
	}
	return nil
}

// convertToJUnit transforms a TestNG report into a JUnit report.
func convertToJUnit(testNG TestNGReport) (JUnitReport, error) {
	var junit JUnitReport

	for _, suite := range testNG.Suites {
		duration, _ := strconv.ParseFloat(suite.Duration, 64)
		durationSec := duration / 1000

		junitSuite := JUnitSuite{
			Name:     suite.Name,
			Tests:    suite.Tests,
			Failures: suite.Failures,
			Skipped:  suite.Skipped,
			Time:     strconv.FormatFloat(durationSec, 'f', 3, 64),
		}

		for _, class := range suite.Classes {
			for _, test := range class.Tests {
				junitCase := JUnitCase{
					Name:      test.Name,
					ClassName: class.Name,
					Duration:  test.Duration,
				}

				if test.Status == "FAIL" {
					junitCase.Failure = &Failure{
						Message:    test.Exception,
						Type:       "Failure",
						StackTrace: test.StackTrace,
					}
				} else if test.Status == "SKIP" {
					junitCase.Skipped = &Skipped{}
				}

				junitSuite.Cases = append(junitSuite.Cases, junitCase)
			}
		}

		junit.Suites = append(junit.Suites, junitSuite)
	}

	return junit, nil
}
