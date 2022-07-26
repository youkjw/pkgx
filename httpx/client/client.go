package client

import (
	"bytes"
	"context"
	"crypto/tls"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/url"
	"strings"
	"time"
)

const (
	httpClientTimeout = 3 * time.Second
	httpContentType   = "application/json"
)

// ClientOption is HTTP client option.
type ClientOption func(*clientOptions)

type clientOptions struct {
	tlsConf  *tls.Config
	timeOut  time.Duration
	endPoint string
}

type Client struct {
	opts   *clientOptions
	cc     *http.Client
	target *Target
}

func WithTlsConf(t *tls.Config) ClientOption {
	return func(options *clientOptions) {
		options.tlsConf = t
	}
}

func WithTimeOut(timeout time.Duration) ClientOption {
	return func(options *clientOptions) {
		options.timeOut = timeout
	}
}

func WithEndpoint(endpoint string) ClientOption {
	return func(options *clientOptions) {
		options.endPoint = endpoint
	}
}

func NewClient(opts ...ClientOption) (*Client, error) {
	opt := &clientOptions{
		timeOut: httpClientTimeout,
	}
	for _, o := range opts {
		o(opt)
	}

	isSecure := opt.tlsConf != nil
	target, err := parseTarget(opt.endPoint, isSecure)
	if err != nil {
		return nil, err
	}
	client := &http.Client{
		Timeout: opt.timeOut,
	}
	return &Client{
		opts:   opt,
		cc:     client,
		target: target,
	}, nil
}

func (client *Client) Do(ctx context.Context, method string, path string, data []byte, opts ...CallOption) ([]byte, error) {
	var (
		contentType = httpContentType
	)
	body := bytes.NewReader(data)
	link := fmt.Sprintf("%s://%s%s", client.target.Scheme, client.target.Host, path)
	req, err := http.NewRequest(method, link, body)
	if err != nil {
		return nil, err
	}

	req.Header.Set("Content-Type", contentType)
	for _, o := range opts {
		err = o.Header(&req.Header)
		if err != nil {
			return nil, err
		}
	}
	return client.do(ctx, req, opts...)
}

func (client *Client) do(ctx context.Context, req *http.Request, opts ...CallOption) ([]byte, error) {
	req = req.WithContext(ctx)
	res, err := client.cc.Do(req)
	if err != nil {
		return nil, err
	}
	defer res.Body.Close()
	if res != nil {
		for _, o := range opts {
			o.After(res)
		}
	}
	body, err := ioutil.ReadAll(res.Body)
	if err != nil {
		return nil, err
	}
	return body, nil
}

type Target struct {
	Scheme string
	Host   string
	Port   string
}

func parseTarget(endpoint string, insecure bool) (*Target, error) {
	if !strings.Contains(endpoint, "://") {
		if insecure {
			endpoint = fmt.Sprintf("%s://%s", "https", endpoint)
		} else {
			endpoint = fmt.Sprintf("%s://%s", "http", endpoint)
		}
	}

	u, err := url.Parse(endpoint)
	if err != nil {
		return nil, err
	}
	target := &Target{
		Scheme: u.Scheme,
		Host:   u.Host,
		Port:   u.Port(),
	}
	return target, nil
}
