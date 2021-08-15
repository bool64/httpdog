package httpdog_test

import (
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/bool64/httpdog"
	"github.com/cucumber/godog"
	"github.com/stretchr/testify/assert"
	"github.com/swaggest/rest/resttest"
)

func TestLocal_RegisterSteps(t *testing.T) {
	mock, srvURL := resttest.NewServerMock()
	mock.OnError = func(err error) {
		assert.NoError(t, err)
	}

	defer mock.Close()

	concurrencyLevel := 5
	setExpectations(mock, concurrencyLevel)

	local := httpdog.NewLocal(srvURL)
	local.OnError = func(err error) {
		assert.NoError(t, err)
	}

	local.Headers = map[string]string{
		"X-Foo": "bar",
	}
	local.ConcurrencyLevel = concurrencyLevel

	suite := godog.TestSuite{
		ScenarioInitializer: func(s *godog.ScenarioContext) {
			local.RegisterSteps(s)
		},
		Options: &godog.Options{
			Format: "pretty",
			Strict: true,
			Paths:  []string{"_testdata/Local.feature"},
		},
	}

	if suite.Run() != 0 {
		t.Fatal("test failed")
	}

	assert.NoError(t, mock.ExpectationsWereMet())
}

func setExpectations(mock *resttest.ServerMock, concurrencyLevel int) {
	mock.Expect(resttest.Expectation{
		Method:       http.MethodGet,
		RequestURI:   "/get-something?foo=bar",
		ResponseBody: []byte(`[{"some":"json"}]`),
		ResponseHeader: map[string]string{
			"Content-Type": "application/json",
		},
	})

	mock.Expect(resttest.Expectation{
		Method:     http.MethodDelete,
		RequestURI: "/bad-request",
		RequestHeader: map[string]string{
			"X-Foo": "bar",
		},
		RequestCookie: map[string]string{
			"c1": "v1",
			"c2": "v2",
		},
		ResponseBody: []byte(`{"error":"oops"}`),
		Status:       http.StatusBadRequest,
	})

	mock.Expect(resttest.Expectation{
		Method:     http.MethodPost,
		RequestURI: "/with-body",
		RequestHeader: map[string]string{
			"X-Foo": "bar",
		},
		RequestBody:  []byte(`[{"some":"json"}]`),
		ResponseBody: []byte(`{"status":"ok"}`),
		ResponseHeader: map[string]string{
			"Content-Type": "application/json",
		},
	})

	del := resttest.Expectation{
		Method:     http.MethodDelete,
		RequestURI: "/delete-something",
		Status:     http.StatusNoContent,
		ResponseHeader: map[string]string{
			"Content-Type": "application/json",
		},
	}

	// Expecting 2 similar requests.
	mock.Expect(del)
	mock.Expect(del)

	// Due to idempotence testing several more requests should be expected.
	delNotFound := del
	delNotFound.Status = http.StatusNotFound
	delNotFound.ResponseBody = []byte(`{"status":"failed"}`)

	for i := 0; i < concurrencyLevel-1; i++ {
		mock.Expect(delNotFound)
	}

	// Expecting request containing json5 comments
	mock.Expect(resttest.Expectation{
		Method:     http.MethodPost,
		RequestURI: "/with-json5-body",
		RequestHeader: map[string]string{
			"X-Foo": "bar",
		},
		RequestBody:  []byte(`[{"some":"json5"}]`),
		ResponseBody: []byte(`{"status":"ok"}`),
		ResponseHeader: map[string]string{
			"Content-Type": "application/json",
		},
	})

	// Expecting request does not contain a valid json
	mock.Expect(resttest.Expectation{
		Method:       http.MethodGet,
		RequestURI:   "/with-csv-body",
		RequestBody:  []byte(`a,b,c`),
		ResponseBody: []byte(`a,b,c`),
	})
}

func TestLocal_RegisterSteps_unexpectedOtherResp(t *testing.T) {
	mock, srvURL := resttest.NewServerMock()
	mock.OnError = func(err error) {
		assert.NoError(t, err)
	}

	defer mock.Close()

	concurrencyLevel := 5
	del := resttest.Expectation{
		Method:     http.MethodDelete,
		RequestURI: "/delete-something",
		Status:     http.StatusNoContent,
		ResponseHeader: map[string]string{
			"Content-Type": "application/json",
		},
	}

	mock.Expect(del)

	// Due to idempotence testing several more requests should be expected.
	delNotFound := del
	delNotFound.Status = http.StatusNotFound
	delNotFound.ResponseBody = []byte(`{"status":"failed"}`)

	for i := 0; i < concurrencyLevel-1; i++ {
		mock.Expect(delNotFound)
	}

	local := httpdog.NewLocal(srvURL)
	errReceived := false

	local.OnError = func(err error) {
		errReceived = true

		assert.EqualError(t, err, "no other responses expected: unexpected response status, expected: 204 (No Content), received: 404 (Not Found)")
	}

	local.ConcurrencyLevel = concurrencyLevel

	suite := godog.TestSuite{
		ScenarioInitializer: func(s *godog.ScenarioContext) {
			local.RegisterSteps(s)
		},
		Options: &godog.Options{
			Output: ioutil.Discard,
			Format: "pretty",
			Strict: true,
			Paths:  []string{"_testdata/LocalFail1.feature"},
		},
	}

	if suite.Run() != 0 {
		t.Fatal("test failed")
	}

	assert.NoError(t, mock.ExpectationsWereMet())
	assert.True(t, errReceived)
}

func TestLocal_RegisterSteps_dynamic(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/user" {
			_, err := w.Write([]byte(`{"id":12345,"name": "John Doe","created_at":"any","updated_at": "any"}`))
			assert.NoError(t, err)

			return
		}

		if r.URL.Path == "/order" {
			b, err := ioutil.ReadAll(r.Body)
			assert.NoError(t, err)
			assert.NoError(t, r.Body.Close())

			assert.Equal(t, `{"user_id":12345,"item_name":"Watermelon"}`, string(b))

			_, err = w.Write([]byte(`{"id":54321,"created_at":"any","updated_at": "any","user_id":12345}`))
			assert.NoError(t, err)

			return
		}
	}))
	defer srv.Close()

	local := httpdog.NewLocal(srv.URL)

	suite := godog.TestSuite{
		ScenarioInitializer: func(s *godog.ScenarioContext) {
			local.RegisterSteps(s)
		},
		Options: &godog.Options{
			Format: "pretty",
			Strict: true,
			Paths:  []string{"_testdata/Dynamic.feature"},
		},
	}

	if suite.Run() != 0 {
		t.Fatal("test failed")
	}
}
