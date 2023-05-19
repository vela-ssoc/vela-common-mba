package netutil

import (
	"bytes"
	"compress/flate"
	"compress/gzip"
	"context"
	"encoding/json"
	"io"
	"net/http"
	"time"
)

// HTTPClient HTTP 客户端
type HTTPClient struct {
	cli *http.Client // 底层 http.Client
}

// NewClient 新建客户端
func NewClient(trip ...http.RoundTripper) HTTPClient {
	cli := http.DefaultClient
	if len(trip) > 0 {
		cli = &http.Client{Transport: trip[0]}
	}

	return HTTPClient{
		cli: cli,
	}
}

func (c HTTPClient) Do(req *http.Request) (*http.Response, error) {
	return c.cli.Do(req)
}

// Fetch 发送请求
func (c HTTPClient) Fetch(ctx context.Context, method string, addr string, body io.Reader, header http.Header) (*http.Response, error) {
	return c.fetch(ctx, method, addr, body, header)
}

// JSON 发送 JSON 响应 JSON
func (c HTTPClient) JSON(ctx context.Context, method, addr string, body, resp any, header http.Header) error {
	res, err := c.doJSON(ctx, method, addr, body, header)
	if err != nil {
		return err
	}
	//goland:noinspection GoUnhandledErrorResult
	defer res.Body.Close()

	return json.NewDecoder(res.Body).Decode(resp)
}

// SilentJSON 发送 JSON 请求但不关注响应结果，只关注是否存在错误
func (c HTTPClient) SilentJSON(ctx context.Context, method, addr string, req any, header http.Header) error {
	res, err := c.doJSON(ctx, method, addr, req, header)
	if err == nil {
		_ = res.Body.Close()
	}
	return err
}

// DoJSONTimeout 超时控制发送请求
func (c HTTPClient) DoJSONTimeout(timeout time.Duration, method, addr string, body, resp any, header http.Header) error {
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	err := c.JSON(ctx, method, addr, body, resp, header)
	cancel()
	return err
}

// NewRequest 构造 http.Request
func (c HTTPClient) NewRequest(ctx context.Context, method string, addr string, body io.Reader, header http.Header) (*http.Request, error) {
	if ctx == nil {
		ctx = context.Background()
	}
	req, err := http.NewRequestWithContext(ctx, method, addr, body)
	if err != nil {
		return nil, err
	}
	if len(header) != 0 {
		req.Header = header
	}

	return req, nil
}

func (c HTTPClient) doJSON(ctx context.Context, method, addr string, body any, header http.Header) (*http.Response, error) {
	rd, err := c.toJSON(body)
	if err != nil {
		return nil, err
	}
	if header == nil {
		header = make(http.Header, 4)
	}
	header.Set("Accept", "application/json")
	header.Set("Content-Type", "application/json; charset=utf-8")

	return c.fetch(ctx, method, addr, rd, header)
}

// fetch 发送请求
func (c HTTPClient) fetch(ctx context.Context, method string, addr string, body io.Reader, header http.Header) (*http.Response, error) {
	req, err := c.NewRequest(ctx, method, addr, body, header)
	if err != nil {
		return nil, err
	}

	encoding := "Accept-Encoding"
	if req.Header.Get(encoding) == "" {
		req.Header.Set("Accept-Encoding", "gzip, deflate")
	}
	res, err := c.cli.Do(req)
	if err != nil {
		return nil, err
	}
	resp := res.Body
	if resp != nil && resp != http.NoBody {
		switch res.Header.Get("Content-Encoding") {
		case "gzip":
			gr, ex := gzip.NewReader(resp)
			if ex != nil {
				_ = resp.Close()
				return nil, ex
			}
			res.Body = gr
		case "deflate":
			fr := flate.NewReader(resp)
			res.Body = fr
		}
	}

	code := res.StatusCode
	if code >= http.StatusOK && code < http.StatusBadRequest { // 200 <= code < 400
		return res, nil
	}

	//goland:noinspection GoUnhandledErrorResult
	defer res.Body.Close()
	buf := make([]byte, 1024)
	n, _ := io.ReadFull(res.Body, buf)
	err = &HTTPError{
		Code: code,
		Body: buf[:n],
	}

	return nil, err
}

// toJSON 转为 JSON
func (HTTPClient) toJSON(v any) (io.Reader, error) {
	if v == nil {
		return nil, nil
	}
	buf := new(bytes.Buffer)
	if err := json.NewEncoder(buf).Encode(v); err != nil {
		return nil, err
	}
	return buf, nil
}
