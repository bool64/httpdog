Feature: External Services

  Scenario: Successful Request
    Given "some-service" receives "GET" request "/get-something?foo=bar"

    And "some-service" request includes header "X-Foo: bar"

    And "some-service" response includes header "X-Bar: foo"

    And "some-service" responds with status "OK" and body
    """
    {"key":"value"}
    """

    And "another-service" receives "POST" request "/post-something" with body
    """
    // Could be a JSON5 too.
    {"foo":"bar"}
    """

    And "another-service" responds with status "OK" and body
    """
    {"theFooWas":"bar"}
    """

    And "some-service" responds with status "OK" and body
    """
    "foo"
    """

    # Request with undefined response.
    And "some-service" receives "GET" request "/never-called"

    # Request that will remain unused after scenario.
    And "another-service" receives "POST" request "/post-something" with body from file
    """
    _testdata/sample.json
    """

    And "another-service" responds with status "OK" and body from file
    """
    _testdata/sample.json5
    """

    When I call external services I receive mocked responses