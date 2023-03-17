package arpc

import (
	"context"
	"encoding/json"
	"mime"
	"mime/multipart"
	"net/http"
	"net/url"
	"reflect"
	"time"
)

// Decoder is the request decoder
type Decoder func(*http.Request, any) error

// Encoder is the response encoder
type Encoder func(http.ResponseWriter, *http.Request, any)

// ErrorEncoder is the error response encoder
type ErrorEncoder func(http.ResponseWriter, *http.Request, error)

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

// Validatable interface
type Validatable interface {
	Valid() error
}

type Manager struct {
	Decoder      Decoder
	Encoder      Encoder
	ErrorEncoder ErrorEncoder
	Validate     bool // set to true to validate request after decode using Validatable interface
	onErrorFuncs []func(http.ResponseWriter, *http.Request, any, error)
	onOKFuncs    []func(http.ResponseWriter, *http.Request, any, any)
	WrapError    func(error) error
}

// New creates new arpc manager
func New() *Manager {
	return &Manager{
		Validate: true,
	}
}

func (m *Manager) decoder() Decoder {
	if m.Decoder == nil {
		return m.Decode
	}
	return m.Decoder
}

func (m *Manager) encoder() Encoder {
	if m.Encoder == nil {
		return m.Encode
	}
	return m.Encoder
}

func (m *Manager) errorEncoder() ErrorEncoder {
	if m.ErrorEncoder == nil {
		return m.EncodeError
	}
	return m.ErrorEncoder
}

func (m *Manager) wrapError(err error) error {
	if m.WrapError != nil {
		return m.WrapError(err)
	}
	return err
}

// OnError calls f when error
func (m *Manager) OnError(f func(w http.ResponseWriter, r *http.Request, req any, err error)) {
	m.onErrorFuncs = append(m.onErrorFuncs, f)
}

// OnOK calls f before encode ok response
func (m *Manager) OnOK(f func(w http.ResponseWriter, r *http.Request, req any, res any)) {
	m.onOKFuncs = append(m.onOKFuncs, f)
}

func (m *Manager) Encode(w http.ResponseWriter, r *http.Request, v any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(struct {
		OK     bool `json:"ok"`
		Result any  `json:"result"`
	}{true, v})
}

func (m *Manager) Decode(r *http.Request, v any) error {
	if p, ok := v.(RequestAdapter); ok {
		p.AdaptRequest(r)
	}

	if r.Method == http.MethodPost {
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
		}
	}

	// fallback to request unmarshaler
	if v, ok := v.(RequestUnmarshaler); ok {
		return WrapError(v.UnmarshalRequest(r))
	}

	return ErrUnsupported
}

func (m *Manager) EncodeError(w http.ResponseWriter, r *http.Request, err error) {
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
		OK    bool `json:"ok"`
		Error any  `json:"error"`
	}{false, err})
}

func (m *Manager) NotFound(w http.ResponseWriter, r *http.Request) {
	m.EncodeError(w, r, ErrNotFound)
}

func (m *Manager) NotFoundHandler() http.Handler {
	return http.HandlerFunc(m.NotFound)
}

type mapIndex int

const (
	_                mapIndex = iota
	miContext                 // context.Context
	miRequest                 // *http.Request
	miResponseWriter          // http.ResponseWriter
	miAny                     // any
	miError                   // error
)

const (
	strContext        = "context.Context"
	strRequest        = "*http.Request"
	strResponseWriter = "http.ResponseWriter"
	strError          = "error"
)

func setOrPanic(m map[mapIndex]int, k mapIndex, v int) {
	if _, exists := m[k]; exists {
		panic("arpc: duplicate input type")
	}
	m[k] = v
}

func (m *Manager) encodeAndHookError(w http.ResponseWriter, r *http.Request, req any, err error) {
	err = m.wrapError(err)

	m.errorEncoder()(w, r, err)

	for _, f := range m.onErrorFuncs {
		f(w, r, req, err)
	}
}

func (m *Manager) Handler(f any) http.Handler {
	hasWriter := false

	fv := reflect.ValueOf(f)
	ft := fv.Type()
	if ft.Kind() != reflect.Func {
		panic("arpc: f must be a function")
	}

	// build mapIn
	numIn := ft.NumIn()
	mapIn := make(map[mapIndex]int)
	for i := 0; i < numIn; i++ {
		fi := ft.In(i)

		// assume this is grpc call options
		if fi.Kind() == reflect.Slice && i == numIn-1 {
			numIn--
			break
		}

		switch fi.String() {
		case strContext:
			setOrPanic(mapIn, miContext, i)
		case strRequest:
			setOrPanic(mapIn, miRequest, i)
		case strResponseWriter:
			setOrPanic(mapIn, miResponseWriter, i)
			hasWriter = true
		default:
			setOrPanic(mapIn, miAny, i)
		}
	}

	// build mapOut
	numOut := ft.NumOut()
	mapOut := make(map[mapIndex]int)
	for i := 0; i < numOut; i++ {
		switch ft.Out(i).String() {
		case strError:
			setOrPanic(mapOut, miError, i)
		default:
			setOrPanic(mapOut, miAny, i)
		}
	}

	var (
		infType reflect.Type
		infPtr  bool
	)
	if i, ok := mapIn[miAny]; ok {
		infType = ft.In(i)
		if infType.Kind() == reflect.Ptr {
			infType = infType.Elem()
			infPtr = true
		}
	}

	encoder := m.encoder()
	decoder := m.decoder()

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var (
			req any
			res any
		)

		vIn := make([]reflect.Value, numIn)
		// inject context
		if i, ok := mapIn[miContext]; ok {
			vIn[i] = reflect.ValueOf(r.Context())
		}
		// inject request interface
		if i, ok := mapIn[miAny]; ok {
			rfReq := reflect.New(infType)
			req = rfReq.Interface()
			err := decoder(r, req)
			if err != nil {
				m.encodeAndHookError(w, r, req, err)
				return
			}

			if m.Validate {
				if req, ok := req.(Validatable); ok {
					err = req.Valid()
					if err != nil {
						m.encodeAndHookError(w, r, req, err)
						return
					}
				}
			}
			if infPtr {
				vIn[i] = rfReq
			} else {
				vIn[i] = rfReq.Elem()
			}
		}
		// inject request
		if i, ok := mapIn[miRequest]; ok {
			vIn[i] = reflect.ValueOf(r)
		}
		// inject response writer
		if i, ok := mapIn[miResponseWriter]; ok {
			vIn[i] = reflect.ValueOf(w)
		}

		vOut := fv.Call(vIn)
		// check error
		if i, ok := mapOut[miError]; ok {
			if vErr := vOut[i]; !vErr.IsNil() {
				if err, ok := vErr.Interface().(error); ok && err != nil {
					m.encodeAndHookError(w, r, req, err)
					return
				}
			}
		}

		// check response
		if i, ok := mapOut[miAny]; ok {
			res = vOut[i].Interface()
			encoder(w, r, res)
		} else if !hasWriter {
			res = _empty
			encoder(w, r, res)
		}

		// run ok hooks
		for _, f := range m.onOKFuncs {
			f(w, r, req, res)
		}
	})
}

type MiddlewareContext struct {
	r *http.Request
	w http.ResponseWriter
}

func (ctx *MiddlewareContext) Request() *http.Request {
	return ctx.r
}

func (ctx *MiddlewareContext) ResponseWriter() http.ResponseWriter {
	return ctx.w
}

func (ctx *MiddlewareContext) Deadline() (deadline time.Time, ok bool) {
	return ctx.r.Context().Deadline()
}

func (ctx *MiddlewareContext) Done() <-chan struct{} {
	return ctx.r.Context().Done()
}

func (ctx *MiddlewareContext) Err() error {
	return ctx.r.Context().Err()
}

func (ctx *MiddlewareContext) Value(key interface{}) interface{} {
	return ctx.r.Context().Value(key)
}

func (ctx *MiddlewareContext) SetRequest(r *http.Request) {
	ctx.r = r
}

func (ctx *MiddlewareContext) SetResponseWriter(w http.ResponseWriter) {
	ctx.w = w
}

func (ctx *MiddlewareContext) SetRequestContext(nctx context.Context) {
	ctx.r = ctx.r.WithContext(nctx)
}

type Middleware func(r *MiddlewareContext) error

func (m *Manager) Middleware(f Middleware) func(http.Handler) http.Handler {
	return func(h http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			ctx := MiddlewareContext{r, w}
			err := f(&ctx)
			if err != nil {
				m.encodeAndHookError(ctx.w, ctx.r, nil, err)
				return
			}
			h.ServeHTTP(ctx.w, ctx.r)
		})
	}
}
