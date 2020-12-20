package httpdog

import (
	"errors"
	"fmt"
	"io/ioutil"
	"net/http"
	"strconv"
	"strings"

	"github.com/cucumber/godog"
	"github.com/swaggest/assertjson/json5"
	"github.com/swaggest/rest/resttest"
)

// NewLocal creates an instance of step-driven HTTP client.
func NewLocal(baseURL string) *Local {
	if !strings.HasPrefix(baseURL, "http://") && !strings.HasPrefix(baseURL, "https://") {
		baseURL = "http://" + baseURL
	}

	baseURL = strings.TrimRight(baseURL, "/")

	return &Local{
		Client: resttest.NewClient(baseURL),
	}
}

// Local is step-driven HTTP client for application local HTTP service.
type Local struct {
	*resttest.Client
}

// RegisterSteps adds HTTP server steps to godog scenario context.
//
// Request Setup
//
// Request configuration needs at least HTTP method and URI.
//
//		When I request HTTP endpoint with method "GET" and URI "/get-something?foo=bar"
//
//
// An additional header can be supplied. For multiple headers, call step multiple times.
//
//		And I request HTTP endpoint with header "X-Foo: bar"
//
// Optionally request body can be configured. If body is a valid JSON5 payload, it will be converted to JSON before use.
// Otherwise, body is used as is.
//
//		And I request HTTP endpoint with body
//		"""
//		[
//		 // JSON5 comments are allowed.
//		 {"some":"json"}
//		]
//		"""
//
// Request body can be provided from file.
//
//		And I request HTTP endpoint with body from file
//		"""
//		path/to/file.json5
//		"""
//
// If endpoint is capable of handling duplicated requests, you can check it for idempotency. This would send multiple
// requests simultaneously and check
//   * if all responses are similar or (all successful like GET),
//   * if responses can be grouped into exactly ONE response of a kind
//     and OTHER responses of another kind (one successful, other failed like with POST).
//
// Number of requests can be configured with `Local.ConcurrencyLevel`, default value is 10.
//
//		And I concurrently request idempotent HTTP endpoint
//
//
// Response Expectations
//
// Response expectation has to be configured with at least one step about status, response body or other responses body
// (idempotency mode).
//
// If response body is a valid JSON5 payload, it is converted to JSON before use.
//
// JSON bodies are compared with https://github.com/swaggest/assertjson which allows ignoring differences
// when expected value is set to `"<ignore-diff>"`.
//
//		And I should have response with body
//		"""
//		[
//		 {"some":"json","time":"<ignore-diff>"}
//		]
//		"""
//
// Response body can be provided from file.
//
//		And I should have response with body from file
//		"""
//		path/to/file.json
//		"""
//
// Status can be defined with either phrase or numeric code.
//
//		Then I should have response with status "OK"
//
// In an idempotent mode you can set expectations for statuses of other responses.
//
//		Then I should have response with status "204"
//
//		And I should have other responses with status "Not Found"
//
// And for bodies of other responses.
//
//		And I should have other responses with body
//		"""
//		{"status":"failed"}
//		"""
//
// Which can be defined as files.
//
//		And I should have other responses with body from file
//		"""
//		path/to/file.json
//		"""
func (l *Local) RegisterSteps(s *godog.ScenarioContext) {
	s.BeforeScenario(func(_ *godog.Scenario) {
		l.Reset()

		if l.JSONComparer.Vars != nil {
			l.JSONComparer.Vars.Reset()
		}
	})

	s.Step(`^I request HTTP endpoint with method "([^"]*)" and URI "([^"]*)"$`, l.iRequestWithMethodAndURI)
	s.Step(`^I request HTTP endpoint with body$`, l.iRequestWithBody)
	s.Step(`^I request HTTP endpoint with body from file$`, l.iRequestWithBodyFromFile)
	s.Step(`^I request HTTP endpoint with header "([^"]*): ([^"]*)"$`, l.iRequestWithHeader)

	s.Step(`^I concurrently request idempotent HTTP endpoint$`, l.iRequestWithConcurrency)

	s.Step(`^I should have response with status "([^"]*)"$`, l.iShouldHaveResponseWithStatus)
	s.Step(`^I should have response with body from file$`, l.iShouldHaveResponseWithBodyFromFile)
	s.Step(`^I should have response with body$`, l.iShouldHaveResponseWithBody)

	s.Step(`^I should have other responses with status "([^"]*)"$`, l.iShouldHaveOtherResponsesWithStatus)
	s.Step(`^I should have other responses with body$`, l.iShouldHaveOtherResponsesWithBody)
	s.Step(`^I should have other responses with body from file$`, l.iShouldHaveOtherResponsesWithBodyFromFile)
}

func (l *Local) iRequestWithMethodAndURI(method, uri string) error {
	l.Reset()
	l.WithMethod(method)
	l.WithURI(uri)

	return nil
}

func loadBodyFromFile(filePath string) ([]byte, error) {
	body, err := ioutil.ReadFile(filePath) // nolint:gosec // File inclusion via variable during tests.
	if err != nil {
		return nil, err
	}

	return loadBody(body)
}

func loadBody(body []byte) ([]byte, error) {
	if json5.Valid(body) {
		return json5.Downgrade(body)
	}

	return body, nil
}

func (l *Local) iRequestWithBodyFromFile(filePath *godog.DocString) error {
	body, err := loadBodyFromFile(filePath.Content)

	if err == nil {
		l.WithBody(body)
	}

	return err
}

func (l *Local) iRequestWithBody(bodyDoc *godog.DocString) error {
	body, err := loadBody([]byte(bodyDoc.Content))

	if err == nil {
		l.WithBody(body)
	}

	return err
}

func (l *Local) iRequestWithHeader(key, value string) error {
	l.WithHeader(key, value)

	return nil
}

var (
	errUnknownStatusCode = errors.New("unknown http status")
	errNoMockForService  = errors.New("no mock for service")
	errUndefinedResponse = errors.New("undefined response")
)

func statusCode(statusOrCode string) (int, error) {
	code, err := strconv.ParseInt(statusOrCode, 10, 64)

	if len(statusMap) == 0 {
		initStatusMap()
	}

	if err != nil {
		code = int64(statusMap[statusOrCode])
	}

	if code == 0 {
		return 0, fmt.Errorf("%w: %q", errUnknownStatusCode, statusOrCode)
	}

	return int(code), nil
}

func (l *Local) iShouldHaveOtherResponsesWithStatus(statusOrCode string) error {
	code, err := statusCode(statusOrCode)
	if err != nil {
		return err
	}

	return l.ExpectOtherResponsesStatus(code)
}

func (l *Local) iShouldHaveResponseWithStatus(statusOrCode string) error {
	code, err := statusCode(statusOrCode)
	if err != nil {
		return err
	}

	return l.ExpectResponseStatus(code)
}

func (l *Local) iShouldHaveResponseWithBody(bodyDoc *godog.DocString) error {
	body, err := loadBody([]byte(bodyDoc.Content))
	if err != nil {
		return err
	}

	return l.ExpectResponseBody(body)
}

func (l *Local) iShouldHaveResponseWithBodyFromFile(filePath *godog.DocString) error {
	body, err := loadBodyFromFile(filePath.Content)
	if err != nil {
		return err
	}

	return l.ExpectResponseBody(body)
}

func (l *Local) iShouldHaveOtherResponsesWithBody(bodyDoc *godog.DocString) error {
	body, err := loadBody([]byte(bodyDoc.Content))
	if err != nil {
		return err
	}

	return l.ExpectOtherResponsesBody(body)
}

func (l *Local) iShouldHaveOtherResponsesWithBodyFromFile(filePath *godog.DocString) error {
	body, err := loadBodyFromFile(filePath.Content)
	if err != nil {
		return err
	}

	return l.ExpectOtherResponsesBody(body)
}

func (l *Local) iRequestWithConcurrency() error {
	l.Concurrently()

	return nil
}

var statusMap = map[string]int{}

func initStatusMap() {
	for i := 100; i < 599; i++ {
		status := http.StatusText(i)
		if status != "" {
			statusMap[status] = i
		}
	}
}
