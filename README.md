# Cucumber HTTP steps for Go

[![Build Status](https://github.com/bool64/httpdog/workflows/test-unit/badge.svg)](https://github.com/bool64/httpdog/actions?query=branch%3Amaster+workflow%3Atest-unit)
[![Coverage Status](https://codecov.io/gh/bool64/httpdog/branch/master/graph/badge.svg)](https://codecov.io/gh/bool64/httpdog)
[![GoDevDoc](https://img.shields.io/badge/dev-doc-00ADD8?logo=go)](https://pkg.go.dev/github.com/bool64/httpdog)
[![Time Tracker](https://wakatime.com/badge/github/bool64/httpdog.svg)](https://wakatime.com/badge/github/bool64/httpdog)
![Code lines](https://sloc.xyz/github/bool64/httpdog/?category=code)
![Comments](https://sloc.xyz/github/bool64/httpdog/?category=comments)

This module implements HTTP-related step definitions
for [`github.com/cucumber/godog`](https://github.com/cucumber/godog).

## Steps

### Local Service

Local service can be tested with client request configuration and response expectations.

#### Request Setup

```gherkin
When I request HTTP endpoint with method "GET" and URI "/get-something?foo=bar"
```

An additional header can be supplied. For multiple headers, call step multiple times.

```gherkin
And I request HTTP endpoint with header "X-Foo: bar"
```

An additional cookie can be supplied. For multiple cookies, call step multiple times.

```gherkin
And I request HTTP endpoint with cookie "name: value"
```

Optionally request body can be configured. If body is a valid JSON5 payload, it will be converted to JSON before use.
Otherwise, body is used as is.

```gherkin
And I request HTTP endpoint with body
"""
[
  // JSON5 comments are allowed.
  {"some":"json"}
]
"""
```

Request body can be provided from file.

```gherkin
And I request HTTP endpoint with body from file
"""
path/to/file.json5
"""
```

If endpoint is capable of handling duplicated requests, you can check it for idempotency. This would send multiple
requests simultaneously and check

* if all responses are similar or (all successful like GET)
* if responses can be grouped into exactly ONE response of a kind and OTHER responses of another kind (one successful,
  other failed like with POST).

Number of requests can be configured with `Local.ConcurrencyLevel`, default value is 10.

```gherkin
And I concurrently request idempotent HTTP endpoint
```

#### Response Expectations

Response expectation has to be configured with at least one step about status, response body or other responses body (
idempotency mode).

If response body is a valid JSON5 payload, it is converted to JSON before use.

JSON bodies are compared with [`assertjson`](https://github.com/swaggest/assertjson) which allows ignoring differences
when expected value is set to `"<ignore-diff>"`.

```gherkin
And I should have response with body
"""
[
  {"some":"json","time":"<ignore-diff>"}
]
"""
```

```gherkin
And I should have response with body from file
"""
path/to/file.json
"""
```

Status can be defined with either phrase or numeric code.

```gherkin
Then I should have response with status "OK"
```

```gherkin
Then I should have response with status "204"

And I should have other responses with status "Not Found"
```

In an idempotent mode you can check other responses.

```gherkin
And I should have other responses with body
"""
{"status":"failed"}
"""
```

```gherkin
And I should have other responses with body from file
"""
path/to/file.json
"""
```

Optionally response headers can be asserted.

```gherkin
Then I should have response with header "Content-Type: application/json"

And I should have other responses with header "Content-Type: text/plain"
And I should have other responses with header "X-Header: abc"
```

### External Services

In simple case you can define expected URL and response.

```gherkin
Given "some-service" receives "GET" request "/get-something?foo=bar"

And "some-service" responds with status "OK" and body
"""
{"key":"value"}
"""
```

Or request with body.

```gherkin
And "another-service" receives "POST" request "/post-something" with body
"""
// Could be a JSON5 too.
{"foo":"bar"}
"""
```

Request with body from a file.

```gherkin
And "another-service" receives "POST" request "/post-something" with body from file
"""
_testdata/sample.json
"""
```

Request can expect to have a header.

```gherkin
And "some-service" request includes header "X-Foo: bar"
```

By default, each configured request is expected to be received 1 time. This can be changed to a different number.

```gherkin
And "some-service" request is received 1234 times
```

Or to be unlimited.

```gherkin
And "some-service" request is received several times
```

By default, requests are expected in same sequential order as they are defined. If there is no stable order you can have
an async expectation. Async requests are expected in any order.

```gherkin
And "some-service" request is async
```

Response may have a header.

```gherkin
And "some-service" response includes header "X-Bar: foo"
```

Response must have a status.

```gherkin
And "some-service" responds with status "OK"
```

Response may also have a body.

```gherkin
And "some-service" responds with status "OK" and body
"""
{"key":"value"}
"""
```

```gherkin
And "another-service" responds with status "200" and body from file
"""
_testdata/sample.json5
"""
```

## Example Feature

```gherkin
Feature: Example

  Scenario: Successful GET Request
    Given "template-service" receives "GET" request "/template/hello"

    And "template-service" responds with status "OK" and body
    """
    Hello, %s!
    """

    When I request HTTP endpoint with method "GET" and URI "/?name=Jane"

    Then I should have response with status "OK"

    And I should have response with body
    """
    Hello, Jane!
    """
```