package arpc_test

import (
	"bytes"
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strconv"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/acoshift/arpc/v2"
)

type request struct {
	A int `json:"a"`
	B int `json:"b"`
}

func f1(r *request) int {
	return r.A + r.B
}

func f2() {
}

func TestSuccess(t *testing.T) {
	t.Parallel()

	m := arpc.New()
	h := m.Handler(f1)
	w := httptest.NewRecorder()
	r := httptest.NewRequest("POST", "/", bytes.NewReader([]byte(`{"a": 2, "b": 3}`)))
	r.Header.Set("Content-Type", "application/json")
	h.ServeHTTP(w, r)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.JSONEq(t, `{"ok":true,"result":5}`, w.Body.String())
}

func TestOnOK(t *testing.T) {
	t.Parallel()

	m := arpc.New()
	m.OnOK(func(w http.ResponseWriter, r *http.Request, req, res any) {
		w.Header().Set("Cache-Control", "public, max-age=1")
	})

	t.Run("f1", func(t *testing.T) {
		h := m.Handler(f1)
		w := httptest.NewRecorder()
		r := httptest.NewRequest("POST", "/", bytes.NewReader([]byte(`{"a": 2, "b": 3}`)))
		r.Header.Set("Content-Type", "application/json")
		h.ServeHTTP(w, r)

		assert.Equal(t, http.StatusOK, w.Code)
		assert.JSONEq(t, `{"ok":true,"result":5}`, w.Body.String())
		assert.Equal(t, "public, max-age=1", w.Header().Get("Cache-Control"))
	})

	t.Run("f2", func(t *testing.T) {
		h := m.Handler(f2)
		w := httptest.NewRecorder()
		r := httptest.NewRequest("POST", "/", bytes.NewReader([]byte(`{}`)))
		r.Header.Set("Content-Type", "application/json")
		h.ServeHTTP(w, r)

		assert.Equal(t, http.StatusOK, w.Code)
		assert.JSONEq(t, `{"ok":true,"result":{}}`, w.Body.String())
	})
}

func TestInvalidContentType(t *testing.T) {
	t.Parallel()

	m := arpc.New()
	h := m.Handler(f1)
	w := httptest.NewRecorder()
	r := httptest.NewRequest("POST", "/", bytes.NewReader([]byte(`{"a": 2, "b": 3}`)))
	h.ServeHTTP(w, r)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestNotFound(t *testing.T) {
	t.Parallel()

	m := arpc.New()
	h := m.NotFoundHandler()
	w := httptest.NewRecorder()
	r := httptest.NewRequest("POST", "/", bytes.NewReader([]byte(`{}`)))
	r.Header.Set("Content-Type", "application/json")
	h.ServeHTTP(w, r)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestError(t *testing.T) {
	t.Parallel()

	m := arpc.New()

	t.Run("String", func(t *testing.T) {
		assert.Equal(t, "some error", arpc.NewError("some error").Error())
		assert.Equal(t, "1000 some error", arpc.NewErrorCode("1000", "some error").Error())
	})

	t.Run("Code", func(t *testing.T) {
		assert.Equal(t, "1000", arpc.NewErrorCode("1000", "some error").(*arpc.Error).Code())
	})

	t.Run("Message", func(t *testing.T) {
		assert.Equal(t, "some error", arpc.NewErrorCode("1000", "some error").(*arpc.Error).Message())
	})

	t.Run("Error", func(t *testing.T) {
		err := arpc.NewErrorCode("0001", "some error")

		h := m.Handler(func() error {
			return err
		})
		w := httptest.NewRecorder()
		r := httptest.NewRequest("POST", "/", bytes.NewReader([]byte(`{}`)))
		r.Header.Set("Content-Type", "application/json")
		h.ServeHTTP(w, r)

		assert.Equal(t, http.StatusOK, w.Code)
		assert.JSONEq(t, `{"ok":false,"error":{"code":"0001","message":"some error"}}`, w.Body.String())
	})

	t.Run("CustomError", func(t *testing.T) {
		h := m.Handler(func() error {
			return &customError{"1A475"}
		})
		w := httptest.NewRecorder()
		r := httptest.NewRequest("POST", "/", bytes.NewReader([]byte(`{}`)))
		r.Header.Set("Content-Type", "application/json")
		h.ServeHTTP(w, r)

		assert.Equal(t, http.StatusOK, w.Code)
		assert.JSONEq(t, `{"ok":false,"error":{"code":"1A475"}}`, w.Body.String())
	})

	t.Run("Internal Error", func(t *testing.T) {
		h := m.Handler(func() error {
			return fmt.Errorf("some error")
		})
		w := httptest.NewRecorder()
		r := httptest.NewRequest("POST", "/", bytes.NewReader([]byte(`{}`)))
		r.Header.Set("Content-Type", "application/json")
		h.ServeHTTP(w, r)

		assert.Equal(t, http.StatusInternalServerError, w.Code)
		assert.JSONEq(t, `{"ok":false,"error":{}}`, w.Body.String())
	})
}

type customError struct {
	Code string `json:"code"`
}

func (err *customError) Error() string {
	return fmt.Sprintf("error %s", err.Code)
}

func (err *customError) OKError() {}

type requestWithAdapter struct {
	A int `json:"a"`
}

func (req *requestWithAdapter) AdaptRequest(r *http.Request) {
	if r.Method == "GET" {
		r.ParseForm()
		r.Method = "POST"
		r.Header.Set("Content-Type", "application/x-www-form-urlencoded")
		r.PostForm = r.Form
	}
}

func (req *requestWithAdapter) UnmarshalForm(v url.Values) error {
	var err error
	req.A, err = strconv.Atoi(v.Get("a"))
	if err != nil {
		return fmt.Errorf("invalid a")
	}
	return nil
}

func TestAdapter(t *testing.T) {
	t.Parallel()

	m := arpc.New()
	h := m.Handler(func(req *requestWithAdapter) (*struct{}, error) {
		assert.Equal(t, 1, req.A)
		return new(struct{}), nil
	})

	t.Run("Success", func(t *testing.T) {
		w := httptest.NewRecorder()
		r := httptest.NewRequest("GET", "/?a=1", nil)
		h.ServeHTTP(w, r)

		assert.Equal(t, http.StatusOK, w.Code)
		assert.JSONEq(t, `{"ok":true,"result":{}}`, w.Body.String())
	})

	t.Run("Error", func(t *testing.T) {
		w := httptest.NewRecorder()
		r := httptest.NewRequest("GET", "/?a=p", nil)
		h.ServeHTTP(w, r)

		assert.Equal(t, http.StatusOK, w.Code)
		assert.JSONEq(t, `{"ok":false,"error":{"message":"invalid a"}}`, w.Body.String())
	})
}

func TestManager_WrapError(t *testing.T) {
	t.Parallel()

	m := arpc.New()
	m.WrapError = func(err error) error {
		return arpc.NewErrorCode("1000", err.Error())
	}
	h := m.Handler(func() error {
		return fmt.Errorf("some error")
	})

	w := httptest.NewRecorder()
	r := httptest.NewRequest("POST", "/", nil)
	h.ServeHTTP(w, r)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.JSONEq(t, `{"ok":false,"error":{"code":"1000","message":"some error"}}`, w.Body.String())
}

func TestMiddleware(t *testing.T) {
	t.Parallel()

	m := arpc.New()

	t.Run("Error", func(t *testing.T) {
		runHandler := false
		h := m.Middleware(func(ctx *arpc.MiddlewareContext) error {
			return arpc.NewError("middleware error")
		})(m.Handler(func() {
			runHandler = true
		}))

		w := httptest.NewRecorder()
		r := httptest.NewRequest("POST", "/", nil)
		h.ServeHTTP(w, r)

		assert.False(t, runHandler)
		assert.JSONEq(t, `{"ok":false,"error":{"message":"middleware error"}}`, w.Body.String())
	})

	t.Run("OK", func(t *testing.T) {
		runHandler := false
		h := m.Middleware(func(ctx *arpc.MiddlewareContext) error {
			return nil
		})(m.Handler(func() {
			runHandler = true
		}))

		w := httptest.NewRecorder()
		r := httptest.NewRequest("POST", "/", nil)
		h.ServeHTTP(w, r)

		assert.True(t, runHandler)
		assert.JSONEq(t, `{"ok":true,"result":{}}`, w.Body.String())
	})
}
