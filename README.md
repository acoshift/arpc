# arpc

ARPC is the Acoshift's opinionated HTTP-RPC styled api

## HTTP Status Code

ARPC will response http with only these 3 status codes

- 200 OK
- 400 Bad Request
- 500 Internal Server Error

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

```http
HTTP/1.1 500 Internal Server Error
Content-Type: application/json; charset=utf-8

{
	"ok": false,
	"error": {} // internal error always return empty object
}
```
