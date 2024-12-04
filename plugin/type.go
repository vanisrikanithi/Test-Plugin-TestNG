package plugin

import "encoding/xml"

// TestNGReport represents the structure of a TestNG XML report.
type TestNGReport struct {
	XMLName xml.Name `xml:"testng-results"`
	Suites  []Suite  `xml:"suite"`
}

// Suite represents a TestNG suite.
type Suite struct {
	Name     string  `xml:"name,attr"`
	Duration string  `xml:"duration-ms,attr"`
	Tests    int     `xml:"tests,attr"`
	Failures int     `xml:"failures,attr"`
	Skipped  int     `xml:"skipped,attr"`
	Classes  []Class `xml:"class"`
}

// Class represents a TestNG class.
type Class struct {
	Name  string `xml:"name,attr"`
	Tests []Test `xml:"test-method"`
}

// Test represents a TestNG test or configuration method.
type Test struct {
	Name       string  `xml:"name,attr"`
	ClassName  string  `xml:"class,attr"`
	Status     string  `xml:"status,attr"`
	Duration   string  `xml:"duration-ms,attr"`
	IsConfig   bool    `xml:"is-config,attr"`
	Parameters []Param `xml:"params>param"`
	Exception  string  `xml:"exception>message"`
	StackTrace string  `xml:"exception>full-stacktrace"`
}

// Param represents a parameter passed to a test method.
type Param struct {
	Name  string `xml:"name,attr"`
	Value string `xml:",chardata"`
}

// JUnitReport represents the structure of a JUnit XML report.
type JUnitReport struct {
	XMLName xml.Name     `xml:"testsuites"`
	Suites  []JUnitSuite `xml:"testsuite"`
}

// JUnitSuite represents a JUnit test suite.
type JUnitSuite struct {
	Name     string      `xml:"name,attr"`
	Tests    int         `xml:"tests,attr"`
	Failures int         `xml:"failures,attr"`
	Skipped  int         `xml:"skipped,attr"`
	Time     string      `xml:"time,attr"`
	Cases    []JUnitCase `xml:"testcase"`
}

// JUnitCase represents a JUnit test case.
type JUnitCase struct {
	Name      string   `xml:"name,attr"`
	ClassName string   `xml:"classname,attr"`
	Duration  string   `xml:"time,attr"`
	Failure   *Failure `xml:"failure,omitempty"`
	Skipped   *Skipped `xml:"skipped,omitempty"`
}

// Failure represents a failed test case.
type Failure struct {
	Message    string `xml:"message,attr"`
	Type       string `xml:"type,attr"`
	StackTrace string `xml:",chardata"`
}

// Skipped represents a skipped test case.
type Skipped struct{}
