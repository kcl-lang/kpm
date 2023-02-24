package Fetch

import (
	"encoding/json"
	"github.com/valyala/fasthttp"
	"sync"
)

type EasyJsonSerialization interface {
	MarshalJSON() ([]byte, error)
	UnmarshalJSON(data []byte) error
}
type Client struct {
	proxy   string
	options []Option
}

func NewClient(proxy string, options ...Option) *Client {
	return &Client{
		proxy:   proxy,
		options: options,
	}
}

type Ctx struct {
	req  *fasthttp.Request
	resp *fasthttp.Response
}

var ctxpool = sync.Pool{New: func() any {
	return &Ctx{
		req:  fasthttp.AcquireRequest(),
		resp: fasthttp.AcquireResponse(),
	}
}}

func AcquireCtx() *Ctx {
	return ctxpool.Get().(*Ctx)
}
func ReleaseRequest(ctx *Ctx) {
	ctx.Reset()
	ctxpool.Put(ctx)
}
func (c *Ctx) Reset() {
	c.req.Reset()
	c.resp.Reset()
}

func (c *Client) Json(ctx *Ctx, endpoint string, reqData any, respData any, options ...Option) error {
	ctx.Reset()
	ctx.req.Header.SetMethod("POST")
	ctx.req.SetRequestURI(endpoint)
	if reqData != nil {
		jsonbytes, err := json.Marshal(reqData)
		if err != nil {
			return err
		}
		ctx.req.SetBody(jsonbytes)
	}
	if c.proxy != "" {
		ctx.req.Header.SetHostBytes(ctx.req.URI().Host())
		ctx.req.URI().SetHost(c.proxy)
	}
	for i := 0; i < len(c.options); i++ {
		options[i](ctx.req)
	}
	for i := 0; i < len(options); i++ {
		options[i](ctx.req)
	}
	if err := fasthttp.Do(ctx.req, ctx.resp); err != nil {
		return err
	}
	body, err := ctx.resp.BodyUncompressed()
	if err != nil {
		return err
	}
	err = json.Unmarshal(body, respData)
	if err != nil {
		return err
	}
	return nil
}

type Option func(req *fasthttp.Request)

// UseGetOption 设置GET请求
func UseGetOption(req *fasthttp.Request) {
	req.Header.SetMethod("GET")
}

// UseCompressOption 设置请求优先使用压缩
func UseCompressOption(req *fasthttp.Request) {
	req.Header.Set("Accept-Encoding", "gzip, deflate, br")
}

// SetHostHeadOption 设置host头
func SetHostHeadOption(req *fasthttp.Request) {
	req.Header.SetHostBytes(req.URI().Host())
}

// SetCustomHostHeadOption  设置用户自定义host头
func SetCustomHostHeadOption(host string) Option {
	return func(req *fasthttp.Request) {
		req.Header.SetHost(host)
	}
}

// Json  json请求与响应
func Json(endpoint string, reqData any, respData any, options ...Option) error {
	req := fasthttp.AcquireRequest()
	defer fasthttp.ReleaseRequest(req)
	req.Header.SetMethod("POST")
	req.SetRequestURI(endpoint)
	if reqData != nil {
		jsonbytes, err := json.Marshal(reqData)
		if err != nil {
			return err
		}
		req.SetBody(jsonbytes)
	}
	for i := 0; i < len(options); i++ {
		options[i](req)
	}
	resp := fasthttp.AcquireResponse()
	defer fasthttp.ReleaseResponse(resp)
	if err := fasthttp.Do(req, resp); err != nil {
		return err
	}
	body, err := resp.BodyUncompressed()
	if err != nil {
		return err
	}
	err = json.Unmarshal(body, respData)
	if err != nil {
		return err
	}
	return nil
}

// Text text请求与响应
func Text(endpoint string, reqData string, options ...Option) (string, error) {
	req := fasthttp.AcquireRequest()
	defer fasthttp.ReleaseRequest(req)
	req.Header.SetMethod("POST")
	req.SetRequestURI(endpoint)
	req.SetBodyString(reqData)
	for i := 0; i < len(options); i++ {
		options[i](req)
	}
	resp := fasthttp.AcquireResponse()
	defer fasthttp.ReleaseResponse(resp)
	if err := fasthttp.Do(req, resp); err != nil {
		return "", err
	}
	body, err := resp.BodyUncompressed()
	if err != nil {
		return "", err
	}
	return string(body), nil
}

// Query 查询请求与响应
func Query(endpoint string, reqData *fasthttp.Args, options ...Option) (string, error) {
	req := fasthttp.AcquireRequest()
	defer fasthttp.ReleaseRequest(req)
	req.Header.SetMethod("GET")
	req.SetRequestURI(endpoint)
	if reqData != nil {
		reqData.CopyTo(req.URI().QueryArgs())
	}
	for i := 0; i < len(options); i++ {
		options[i](req)
	}
	resp := fasthttp.AcquireResponse()
	defer fasthttp.ReleaseResponse(resp)
	if err := fasthttp.Do(req, resp); err != nil {
		return "", err
	}
	body, err := resp.BodyUncompressed()
	if err != nil {
		return "", err
	}
	return string(body), nil
}

// EasyJson 使用easyJson接口的json请求与响应
func EasyJson(endpoint string, reqData EasyJsonSerialization, respData EasyJsonSerialization, options ...Option) error {
	jsonbytes, err := reqData.MarshalJSON()
	if err != nil {
		return err
	}
	req := fasthttp.AcquireRequest()
	defer fasthttp.ReleaseRequest(req)
	req.Header.SetMethod("POST")
	req.SetRequestURI(endpoint)
	req.SetBody(jsonbytes)
	for i := 0; i < len(options); i++ {
		options[i](req)
	}
	resp := fasthttp.AcquireResponse()
	defer fasthttp.ReleaseResponse(resp)
	if err = fasthttp.Do(req, resp); err != nil {
		return err
	}
	body, err := resp.BodyUncompressed()
	if err != nil {
		return err
	}
	err = respData.UnmarshalJSON(body)
	if err != nil {
		return err
	}
	return nil
}
