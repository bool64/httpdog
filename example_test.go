package httpdog_test

import (
	"fmt"
	"io/ioutil"
	"net/http"
	"net/http/httptest"

	"github.com/bool64/httpdog"
	"github.com/cucumber/godog"
)

func ExampleNewLocal() {
	external := httpdog.External{}
	templateService := external.Add("template-service")

	h := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		req, _ := http.NewRequest(http.MethodGet, templateService+"/template/hello", nil) // nolint // Handle errors.
		resp, _ := http.DefaultTransport.RoundTrip(req)                                   // nolint // Handle errors.
		tpl, _ := ioutil.ReadAll(resp.Body)                                               // nolint // Handle errors.

		_, _ = w.Write([]byte(fmt.Sprintf(string(tpl), r.URL.Query().Get("name")))) // nolint // Handle errors.
	})

	srv := httptest.NewServer(h)
	defer srv.Close()

	local := httpdog.NewLocal(srv.URL)

	suite := godog.TestSuite{
		ScenarioInitializer: func(s *godog.ScenarioContext) {
			local.RegisterSteps(s)
			external.RegisterSteps(s)
		},
		Options: &godog.Options{
			Format: "pretty",
			Strict: true,
			Paths:  []string{"_testdata/Example.feature"},
			Output: ioutil.Discard,
		},
	}
	status := suite.Run()

	if status != 0 {
		fmt.Println("test failed")
	} else {
		fmt.Println("test passed")
	}

	// Output:
	// test passed
}