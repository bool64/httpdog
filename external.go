package httpdog

import (
	"fmt"

	"github.com/bool64/shared"
	"github.com/cucumber/godog"
	"github.com/swaggest/rest/resttest"
)

// External is a collection of step-driven HTTP servers to serve requests of application with mocked data.
type External struct {
	pending map[string]resttest.Expectation
	mocks   map[string]*resttest.ServerMock

	OnError func(err error)
	Vars    *shared.Vars
}

// RegisterSteps adds steps to godog scenario context to serve outgoing requests with mocked data.
//
// In simple case you can define expected URL and response.
//
//		Given "some-service" receives "GET" request "/get-something?foo=bar"
//
//		And "some-service" responds with status "OK" and body
//		"""
//		{"key":"value"}
//		"""
//
// Or request with body.
//
//		And "another-service" receives "POST" request "/post-something" with body
//		"""
//		// Could be a JSON5 too.
//		{"foo":"bar"}
//		"""
//
// Request with body from a file.
//
//		And "another-service" receives "POST" request "/post-something" with body from file
//		"""
//		_testdata/sample.json
//		"""
//
// Request can expect to have a header.
//
//		And "some-service" request includes header "X-Foo: bar"
//
// By default, each configured request is expected to be received 1 time. This can be changed to a different number.
//
//		And "some-service" request is received 1234 times
//
// Or to be unlimited.
//
//		And "some-service" request is received several times
//
// Response may have a header.
//
//		And "some-service" response includes header "X-Bar: foo"
//
// Response must have status and body (body can be empty).
//
//		And "some-service" responds with status "OK" and body
//		"""
//		{"key":"value"}
//		"""
//
// Response body can also be defined in file.
//
//		And "another-service" responds with status "200" and body from file
//		"""
//		_testdata/sample.json5
//		"""
func (e *External) RegisterSteps(s *godog.ScenarioContext) {
	e.pending = make(map[string]resttest.Expectation, len(e.mocks))

	s.Step(`^"([^"]*)" receives "([^"]*)" request "([^"]*)"$`,
		e.serviceReceivesRequest)
	s.Step(`^"([^"]*)" receives "([^"]*)" request "([^"]*)" with body$`,
		e.serviceReceivesRequestWithBody)
	s.Step(`^"([^"]*)" receives "([^"]*)" request "([^"]*)" with body from file$`,
		e.serviceReceivesRequestWithBodyFromFile)
	s.Step(`^"([^"]*)" request includes header "([^"]*): ([^"]*)"$`,
		e.serviceRequestIncludesHeader)
	s.Step(`^"([^"]*)" request is received several times$`,
		e.serviceReceivesRequestMultipleTimes)
	s.Step(`^"([^"]*)" request is received (\d+) times$`,
		e.serviceReceivesRequestNTimes)

	s.Step(`^"([^"]*)" response includes header "([^"]*): ([^"]*)"$`,
		e.serviceResponseIncludesHeader)
	s.Step(`^"([^"]*)" responds with status "([^"]*)" and body$`,
		e.serviceRespondsWithStatusAndBody)
	s.Step(`^"([^"]*)" responds with status "([^"]*)" and body from file$`,
		e.serviceRespondsWithStatusAndBodyFromFile)

	s.BeforeScenario(func(i *godog.Scenario) {
		for _, mock := range e.mocks {
			mock.ResetExpectations()
		}

		if e.Vars != nil {
			e.Vars.Reset()
		}
	})

	s.AfterScenario(func(s *godog.Scenario, _ error) {
		onError := e.OnError
		if onError == nil {
			onError = func(err error) {
				panic(err)
			}
		}

		if len(e.pending) > 0 {
			for service, req := range e.pending {
				onError(fmt.Errorf("service: %s, %w for: %s %s",
					service, errUndefinedResponse, req.Method, req.RequestURI))
			}
		}

		for service, mock := range e.mocks {
			err := mock.ExpectationsWereMet()
			if err != nil {
				onError(fmt.Errorf("service: %s, scenario: %s, expectations were not met: %w", service, s.Name, err))
			}
		}
	})
}

// GetMock exposes mock of external service.
func (e *External) GetMock(service string) *resttest.ServerMock {
	return e.mocks[service]
}

// Add starts a mocked server for a named service and returns url.
func (e *External) Add(service string, options ...func(mock *resttest.ServerMock)) string {
	mock, url := resttest.NewServerMock()

	mock.OnError = e.OnError
	mock.JSONComparer.Vars = e.Vars

	for _, option := range options {
		option(mock)
	}

	if e.mocks == nil {
		e.mocks = make(map[string]*resttest.ServerMock, 1)
	}

	e.mocks[service] = mock

	return url
}

func (e *External) serviceReceivesRequestWithPreparedBody(service, method, requestURI string, body []byte) error {
	err := e.serviceReceivesRequest(service, method, requestURI)
	if err != nil {
		return err
	}

	pending := e.pending[service]

	pending.RequestBody = body
	e.pending[service] = pending

	return nil
}

func (e *External) serviceRequestIncludesHeader(service, header, value string) error {
	pending := e.pending[service]

	if pending.RequestHeader == nil {
		pending.RequestHeader = make(map[string]string, 1)
	}

	pending.RequestHeader[header] = value
	e.pending[service] = pending

	return nil
}

func (e *External) serviceReceivesRequestWithBody(service, method, requestURI string, bodyDoc *godog.DocString) error {
	body, err := loadBody([]byte(bodyDoc.Content))
	if err != nil {
		return err
	}

	return e.serviceReceivesRequestWithPreparedBody(service, method, requestURI, body)
}

func (e *External) serviceReceivesRequestWithBodyFromFile(service, method, requestURI string, filePath *godog.DocString) error {
	body, err := loadBodyFromFile(filePath.Content)
	if err != nil {
		return err
	}

	return e.serviceReceivesRequestWithPreparedBody(service, method, requestURI, body)
}

func (e *External) serviceReceivesRequest(service, method, requestURI string) error {
	if _, ok := e.mocks[service]; !ok {
		return fmt.Errorf("%w: %q", errNoMockForService, service)
	}

	pending := e.pending[service]
	pending.Method = method
	pending.RequestURI = requestURI
	e.pending[service] = pending

	return nil
}

func (e *External) serviceReceivesRequestNTimes(service string, n int) error {
	if _, ok := e.mocks[service]; !ok {
		return fmt.Errorf("%w: %q", errNoMockForService, service)
	}

	pending := e.pending[service]
	pending.Repeated = n
	e.pending[service] = pending

	return nil
}

func (e *External) serviceReceivesRequestMultipleTimes(service string) error {
	if _, ok := e.mocks[service]; !ok {
		return fmt.Errorf("%w: %q", errNoMockForService, service)
	}

	pending := e.pending[service]
	pending.Unlimited = true
	e.pending[service] = pending

	return nil
}

func (e *External) serviceRespondsWithStatusAndPreparedBody(service, statusOrCode string, body []byte) error {
	m, ok := e.mocks[service]
	if !ok {
		return fmt.Errorf("%w: %q", errNoMockForService, service)
	}

	code, err := statusCode(statusOrCode)
	if err != nil {
		return err
	}

	pending := e.pending[service]
	delete(e.pending, service)

	pending.Status = code
	pending.ResponseBody = body

	if pending.ResponseHeader == nil {
		pending.ResponseHeader = map[string]string{}
	}

	m.Expect(pending)

	return nil
}

func (e *External) serviceResponseIncludesHeader(service, header, value string) error {
	_, ok := e.mocks[service]
	if !ok {
		return fmt.Errorf("%w: %q", errNoMockForService, service)
	}

	pending := e.pending[service]
	if pending.ResponseHeader == nil {
		pending.ResponseHeader = make(map[string]string, 1)
	}

	pending.ResponseHeader[header] = value
	e.pending[service] = pending

	return nil
}

func (e *External) serviceRespondsWithStatusAndBody(service, statusOrCode string, bodyDoc *godog.DocString) error {
	body, err := loadBody([]byte(bodyDoc.Content))
	if err != nil {
		return err
	}

	return e.serviceRespondsWithStatusAndPreparedBody(service, statusOrCode, body)
}

func (e *External) serviceRespondsWithStatusAndBodyFromFile(service, statusOrCode string, filePath *godog.DocString) error {
	body, err := loadBodyFromFile(filePath.Content)
	if err != nil {
		return err
	}

	return e.serviceRespondsWithStatusAndPreparedBody(service, statusOrCode, body)
}
