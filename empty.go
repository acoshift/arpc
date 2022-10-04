package arpc

import (
	"mime/multipart"
	"net/http"
	"net/url"
)

type Empty struct{}

func (Empty) UnmarshalForm(v url.Values) error               { return nil }
func (Empty) UnmarshalMultipartForm(v *multipart.Form) error { return nil }
func (Empty) UnmarshalRequest(r *http.Request) error         { return nil }

var _empty = &Empty{}
