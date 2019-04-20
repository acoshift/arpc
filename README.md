# arpc

ARPC is the Acoshift's opinionated HTTP-RPC styled api

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

### Method not allowed

- Developer (api caller) send invalid method

```http
HTTP/1.1 400 Bad Request
Content-Type: application/json; charset=utf-8

{
	"ok": false,
	"error": {
		"message": "method not allowed"
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
