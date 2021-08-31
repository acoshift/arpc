package arpc

import (
	"encoding/json"
	"mime"
	"mime/multipart"
	"net/http"
	"net/url"

	"github.com/acoshift/hrpc/v3"
)

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

type Manager struct {
	m            hrpc.Manager
	onErrorFuncs []func(http.ResponseWriter, *http.Request, error)
	onOKFuncs    []func(http.ResponseWriter, *http.Request, interface{})
}

// New creates new arpc manager
func New() *Manager {
	var m Manager
	m.m = hrpc.Manager{
		Decoder:      m.decoder,
		Encoder:      m.encoder,
		ErrorEncoder: m.errorEncoder,
	}
	return &m
}

// SetValidate sets hrpc manager validate state
func (m *Manager) SetValidate(enable bool) {
	m.m.Validate = enable
}

// OnError calls f when error
func (m *Manager) OnError(f func(w http.ResponseWriter, r *http.Request, err error)) {
	m.onErrorFuncs = append(m.onErrorFuncs, f)
}

// OnOK calls f before encode ok response
func (m *Manager) OnOK(f func(w http.ResponseWriter, r *http.Request, v interface{})) {
	m.onOKFuncs = append(m.onOKFuncs, f)
}

func (m *Manager) encoder(w http.ResponseWriter, r *http.Request, v interface{}) {
	for _, f := range m.onOKFuncs {
		f(w, r, v)
	}
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(struct {
		OK     bool        `json:"ok"`
		Result interface{} `json:"result"`
	}{true, v})
}

func (m *Manager) decoder(r *http.Request, v interface{}) error {
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

func (m *Manager) errorEncoder(w http.ResponseWriter, r *http.Request, err error) {
	for _, f := range m.onErrorFuncs {
		f(w, r, err)
	}

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
func (m *Manager) Handler(f interface{}) http.Handler {
	return m.m.Handler(f)
}

func (m *Manager) NotFound(w http.ResponseWriter, r *http.Request) {
	m.errorEncoder(w, r, errNotFound)
}

func (m *Manager) NotFoundHandler() http.Handler {
	return http.HandlerFunc(m.NotFound)
}
