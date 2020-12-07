package httpdog_test

import (
	"bytes"
	"io/ioutil"
	"net/http"
	"testing"
	"time"

	"github.com/bool64/httpdog"
	"github.com/cucumber/godog"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/swaggest/assertjson"
)

func TestRegisterExternal(t *testing.T) {
	es := httpdog.External{}

	errs := []string{}
	es.OnError = func(err error) {
		errs = append(errs, err.Error())
	}

	someServiceURL := es.Add("some-service")
	anotherServiceURL := es.Add("another-service")

	suite := godog.TestSuite{
		ScenarioInitializer: func(s *godog.ScenarioContext) {
			es.RegisterSteps(s)

			s.Step(`^I call external services I receive mocked responses$`,
				callServices(t, someServiceURL, anotherServiceURL))
		},
		Options: &godog.Options{
			Format:    "pretty",
			Strict:    true,
			Paths:     []string{"_testdata/External.feature"},
			Randomize: time.Now().UTC().UnixNano(),
		},
	}
	status := suite.Run()

	if status != 0 {
		t.Fatal("unexpected error")
	}

	assert.Equal(t, []string{
		"service: some-service, undefined response for: GET /never-called",
		"service: another-service, scenario: Successful Request, expectations were not met: " +
			"there are remaining expectations that were not met: POST /post-something",
	}, errs)
}

func callServices(t *testing.T, someServiceURL, anotherServiceURL string) func() error {
	return func() error {
		// Hitting `"some-service" receives "GET" request "/get-something?foo=bar"`.
		req, err := http.NewRequest(http.MethodGet, someServiceURL+"/get-something?foo=bar", nil)
		require.NoError(t, err)

		req.Header.Set("X-Foo", "bar")

		resp, err := http.DefaultTransport.RoundTrip(req)
		require.NoError(t, err)

		assert.Equal(t, "foo", resp.Header.Get("X-Bar"))

		respBody, err := ioutil.ReadAll(resp.Body)
		require.NoError(t, resp.Body.Close())
		require.NoError(t, err)

		assertjson.Equal(t, []byte(`{"key":"value"}`), respBody, string(respBody))

		// Hitting `"another-service" receives "POST" request "/post-something" with body`.
		req, err = http.NewRequest(http.MethodPost, anotherServiceURL+"/post-something", bytes.NewReader([]byte(`{"foo":"bar"}`)))
		require.NoError(t, err)

		resp, err = http.DefaultTransport.RoundTrip(req)
		require.NoError(t, err)

		respBody, err = ioutil.ReadAll(resp.Body)
		require.NoError(t, resp.Body.Close())
		require.NoError(t, err)

		assertjson.Equal(t, []byte(`{"theFooWas":"bar"}`), respBody)

		// Hitting `"some-service" responds with status "OK" and body`.
		req, err = http.NewRequest(http.MethodGet, someServiceURL+"/does-not-matter", nil)
		require.NoError(t, err)

		resp, err = http.DefaultTransport.RoundTrip(req)
		require.NoError(t, err)

		respBody, err = ioutil.ReadAll(resp.Body)
		require.NoError(t, resp.Body.Close())
		require.NoError(t, err)

		assertjson.Equal(t, []byte(`"foo"`), respBody)

		return nil
	}
}
