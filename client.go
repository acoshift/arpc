package arpc

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"net/http"
	"strings"
)

func Invoke[P, R any](ctx context.Context, client *http.Client, endpoint, method string, p *P) (*R, error) {
	var reqBuff bytes.Buffer
	err := json.NewEncoder(&reqBuff).Encode(p)
	if err != nil {
		return nil, err
	}

	if !strings.HasSuffix(endpoint, "/") && !strings.HasPrefix(method, "/") {
		endpoint += "/"
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint+method, &reqBuff)
	if err != nil {
		return nil, err
	}

	if client == nil {
		client = http.DefaultClient
	}
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	defer io.Copy(io.Discard, resp.Body)

	var res struct {
		OK    bool `json:"ok"`
		Error struct {
			Code    string `json:"code"`
			Message string `json:"message"`
		} `json:"error"`
		Result R `json:"result"`
	}
	err = json.NewDecoder(resp.Body).Decode(&res)
	if err != nil {
		return nil, err
	}
	if !res.OK {
		if resp.StatusCode == http.StatusInternalServerError {
			return nil, internalError{}
		}
		if resp.StatusCode == http.StatusBadRequest {
			return nil, NewProtocolError(res.Error.Code, res.Error.Message)
		}
		return nil, NewErrorCode(res.Error.Code, res.Error.Message)
	}
	return &res.Result, nil
}
