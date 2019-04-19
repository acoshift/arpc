package arpc

import (
	"encoding/json"
	"mime"
	"mime/multipart"
	"net/http"
	"net/url"

	"github.com/acoshift/hrpc"
)

var m = hrpc.Manager{
	Encoder:      encoder,
	Decoder:      decoder,
	ErrorEncoder: errorEncoder,
	Validate:     true,
}

// Error type
type Error struct {
	Status  int    `json:"-"`
	Message string `json:"message"`
}

func (err *Error) Error() string {
	return err.Message
}

var (
	ErrUnsupported error = &Error{http.StatusUnsupportedMediaType, "unsupported content type"}

	errMethodNotAllowed error = &Error{http.StatusMethodNotAllowed, "method not allowed"}
	errNotFound         error = &Error{http.StatusNotFound, "not found"}
)

func encoder(w http.ResponseWriter, r *http.Request, v interface{}) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(struct {
		OK     bool        `json:"ok"`
		Result interface{} `json:"result"`
	}{true, v})
}

// FormUnmarshaler interface
type FormUnmarshaler interface {
	UnmarshalForm(v url.Values) error
}

// MultipartFormUnmarshaler interface
type MultipartFormUnmarshaler interface {
	UnmarshalMultipartForm(v *multipart.Form) error
}

// RequestAdapter converts request to arpc before decode
type RequestAdapter interface {
	AdaptRequest(r *http.Request)
}

func badRequestError(err error) error {
	if err == nil {
		return nil
	}
	if _, ok := err.(*Error); ok {
		return err
	}
	return &Error{Status: http.StatusBadRequest, Message: err.Error()}
}

func decoder(r *http.Request, v interface{}) error {
	if v, ok := v.(RequestAdapter); ok {
		v.AdaptRequest(r)
	}

	if r.Method != http.MethodPost {
		return errMethodNotAllowed
	}

	mt, _, _ := mime.ParseMediaType(r.Header.Get("Content-Type"))
	switch mt {
	case "application/json":
		return json.NewDecoder(r.Body).Decode(v)
	case "application/x-www-form-urlencoded":
		err := r.ParseForm()
		if err != nil {
			return badRequestError(err)
		}
		if v, ok := v.(FormUnmarshaler); ok {
			return badRequestError(v.UnmarshalForm(r.PostForm))
		}
	case "multipart/form-data":
		err := r.ParseMultipartForm(32 << 20)
		if err != nil {
			return badRequestError(err)
		}
		if v, ok := v.(MultipartFormUnmarshaler); ok {
			return badRequestError(v.UnmarshalMultipartForm(r.MultipartForm))
		}
	}
	return ErrUnsupported
}

func errorEncoder(w http.ResponseWriter, r *http.Request, err error) {
	status := http.StatusInternalServerError
	if err, ok := err.(*Error); ok {
		status = err.Status
	}

	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(struct {
		OK    bool  `json:"ok"`
		Error error `json:"error"`
	}{false, err})
}

// Handler converts f to handler
func Handler(f interface{}) http.Handler {
	return m.Handler(f)
}

func NotFound(w http.ResponseWriter, r *http.Request) {
	errorEncoder(w, r, errNotFound)
}

func NotFoundHandler() http.Handler {
	return http.HandlerFunc(NotFound)
}
