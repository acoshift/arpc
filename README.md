# arpc

![Build Status](https://github.com/acoshift/arpc/actions/workflows/test.yaml/badge.svg?branch=master)
[![codecov](https://codecov.io/gh/acoshift/arpc/branch/master/graph/badge.svg?token=3AMuow5vXh)](https://codecov.io/gh/acoshift/arpc)
[![Go Report Card](https://goreportcard.com/badge/github.com/acoshift/arpc)](https://goreportcard.com/report/github.com/acoshift/arpc)
[![GoDoc](https://pkg.go.dev/badge/github.com/acoshift/arpc)](https://pkg.go.dev/github.com/acoshift/arpc)

ARPC is the Acoshift's opinionated HTTP-RPC styled api

## Installation

```
go get -u github.com/acoshift/arpc/v2
```

## HTTP Status Code

ARPC will response http with only these 3 status codes

- 200 OK - function works as expected
- 400 Bad Request - developer (api caller) error, should never happened in production
- 500 Internal Server Error - server error, should never happened (server broken)

## Example Responses

### Success Result

```http
HTTP/1.1 200 OK
Content-Type: application/json; charset=utf-8

{
	"ok": true,
	"result": {
		// result object
	}
}
```

### Error Result

- Validate error
- Precondition failed
- User error

```http
HTTP/1.1 200 OK
Content-Type: application/json; charset=utf-8

{
	"ok": false,
	"error": {
		"message": "some error message"
	}
}
```

### Function not found

- Developer (api caller) call not exists function

```http
HTTP/1.1 400 Bad Request
Content-Type: application/json; charset=utf-8

{
	"ok": false,
	"error": {
		"message": "not found"
	}
}
```

### Unsupported Content-Type

- Developer (api caller) send invalid content type

```http
HTTP/1.1 400 Bad Request
Content-Type: application/json; charset=utf-8

{
	"ok": false,
	"error": {
		"message": "unsupported content type"
	}
}
```

### Internal Server Error

- Server broken !!!

```http
HTTP/1.1 500 Internal Server Error
Content-Type: application/json; charset=utf-8

{
	"ok": false,
	"error": {} // internal error always return empty object
}
```
