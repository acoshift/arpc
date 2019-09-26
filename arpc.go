package arpc

import (
	"encoding/json"
	"mime"
	"mime/multipart"
	"net/http"
	"net/url"

	"github.com/acoshift/hrpc/v3"
)

var m = hrpc.Manager{
	Encoder:      encoder,
	Decoder:      decoder,
	ErrorEncoder: errorEncoder,
}

// SetValidate sets hrpc manager validate state
func SetValidate(enable bool) {
	m.Validate = enable
}

func encoder(w http.ResponseWriter, _ *http.Request, v interface{}) {
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

// RequestUnmarshaler interface
type RequestUnmarshaler interface {
	UnmarshalRequest(r *http.Request) error
}

// RequestAdapter converts request to arpc before decode
type RequestAdapter interface {
	AdaptRequest(r *http.Request)
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
		return WrapError(json.NewDecoder(r.Body).Decode(v))
	case "application/x-www-form-urlencoded":
		err := r.ParseForm()
		if err != nil {
			return WrapError(err)
		}
		if v, ok := v.(FormUnmarshaler); ok {
			return WrapError(v.UnmarshalForm(r.PostForm))
		}
	case "multipart/form-data":
		err := r.ParseMultipartForm(32 << 20)
		if err != nil {
			return WrapError(err)
		}
		if v, ok := v.(MultipartFormUnmarshaler); ok {
			return WrapError(v.UnmarshalMultipartForm(r.MultipartForm))
		}
	default:
		// fallback to request unmarshaler
		if v, ok := v.(RequestUnmarshaler); ok {
			return WrapError(v.UnmarshalRequest(r))
		}
	}
	return ErrUnsupported
}

func errorEncoder(w http.ResponseWriter, r *http.Request, err error) {
	var status int
	switch err.(type) {
	case OKError:
		status = http.StatusOK
	case *ProtocolError:
		status = http.StatusBadRequest
	default:
		status = http.StatusInternalServerError
		err = internalError{}
	}

	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(struct {
		OK    bool        `json:"ok"`
		Error interface{} `json:"error"`
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
