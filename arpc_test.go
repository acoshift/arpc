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

	"github.com/acoshift/arpc"
)

type request struct {
	A int `json:"a"`
	B int `json:"b"`
}

func sum(r *request) int {
	return r.A + r.B
}

func div(r *request) (int, error) {
	if r.B == 0 {
		return 0, fmt.Errorf("divide by zero")
	}
	return r.A / r.B, nil
}

func TestSuccess(t *testing.T) {
	t.Parallel()

	h := arpc.Handler(sum)
	w := httptest.NewRecorder()
	r := httptest.NewRequest("POST", "/", bytes.NewReader([]byte(`{"a": 2, "b": 3}`)))
	r.Header.Set("Content-Type", "application/json")
	h.ServeHTTP(w, r)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.JSONEq(t, `{"ok":true,"result":5}`, w.Body.String())
}

func TestInvalidContentType(t *testing.T) {
	t.Parallel()

	h := arpc.Handler(sum)
	w := httptest.NewRecorder()
	r := httptest.NewRequest("POST", "/", bytes.NewReader([]byte(`{"a": 2, "b": 3}`)))
	h.ServeHTTP(w, r)

	assert.Equal(t, http.StatusUnsupportedMediaType, w.Code)
}

func TestInvalidMethod(t *testing.T) {
	t.Parallel()

	h := arpc.Handler(sum)
	w := httptest.NewRecorder()
	r := httptest.NewRequest("GET", "/", nil)
	h.ServeHTTP(w, r)

	assert.Equal(t, http.StatusMethodNotAllowed, w.Code)
}

func TestNotFound(t *testing.T) {
	t.Parallel()

	h := arpc.NotFoundHandler()
	w := httptest.NewRecorder()
	r := httptest.NewRequest("POST", "/", bytes.NewReader([]byte(`{}`)))
	r.Header.Set("Content-Type", "application/json")
	h.ServeHTTP(w, r)

	assert.Equal(t, http.StatusNotFound, w.Code)
}

func TestError(t *testing.T) {
	var err error = &arpc.Error{Status: http.StatusInternalServerError, Message: "internal error"}
	assert.Equal(t, "internal error", err.Error())

	h := arpc.Handler(func() error {
		return err
	})
	w := httptest.NewRecorder()
	r := httptest.NewRequest("POST", "/", bytes.NewReader([]byte(`{}`)))
	r.Header.Set("Content-Type", "application/json")
	h.ServeHTTP(w, r)

	assert.Equal(t, http.StatusInternalServerError, w.Code)
	assert.JSONEq(t, `{"ok":false,"error":{"message":"internal error"}}`, w.Body.String())
}

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

	h := arpc.Handler(func(req *requestWithAdapter) (*struct{}, error) {
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

		assert.Equal(t, http.StatusBadRequest, w.Code)
		assert.JSONEq(t, `{"ok":false,"error":{"message":"invalid a"}}`, w.Body.String())
	})
}
