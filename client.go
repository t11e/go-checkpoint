package checkpoint

import (
	"context"
	"net/http"
	"net/url"
	"time"

	"go.uber.org/zap"
)

type Option func(*Client) error

func WithHTTPClient(hc *http.Client) Option {
	return func(c *Client) error {
		c.hc = hc
		return nil
	}
}

func WithLogger(logger *zap.SugaredLogger) Option {
	return func(c *Client) error {
		c.logger = logger.Named("ingest")
		return nil
	}
}

func WithHost(host string) Option {
	return func(c *Client) error {
		c.host = host
		return nil
	}
}

func WithScheme(scheme string) Option {
	return func(c *Client) error {
		c.scheme = scheme
		return nil
	}
}

func WithSession(session string) Option {
	return func(c *Client) error {
		c.session = session
		return nil
	}
}

type Client struct {
	hc      *http.Client
	scheme  string
	host    string
	session string
	logger  *zap.SugaredLogger
}

func New(opts ...Option) (*Client, error) {
	c := Client{
		hc:     http.DefaultClient,
		scheme: "http",
		logger: zap.NewNop().Sugar(),
	}
	for _, o := range opts {
		if err := o(&c); err != nil {
			return nil, err
		}
	}
	c.logger = c.logger.Named("checkpoint").With("service", c.scheme+"://"+c.host)
	return &c, nil
}

func (c *Client) WithSession(session string) *Client {
	client := *c
	client.session = session
	return &client
}

type getIdentityResponse struct {
	Identity *Identity `json:"identity"`
	Profile  *Profile  `json:"profile"`
}

func (c *Client) GetCurrentIdentity(ctx context.Context) (*Identity, error) {
	var resp getIdentityResponse
	_, err := c.doGet(ctx, "/identities/me", url.Values{}, &resp)
	if err != nil {
		if isStatus(err, http.StatusPreconditionFailed) {
			return nil, nil
		}
		return nil, err
	}
	return resp.Identity, nil
}

func (c *Client) GetCurrentUser(ctx context.Context) (*Identity, *Profile, error) {
	var resp getIdentityResponse
	_, err := c.doGet(ctx, "/identities/me", url.Values{}, &resp)
	if err != nil {
		if isStatus(err, http.StatusPreconditionFailed) {
			return nil, nil, nil
		}
		return nil, nil, err
	}
	return resp.Identity, resp.Profile, nil
}

func (c *Client) doGet(
	ctx context.Context,
	path string,
	params url.Values,
	output interface{}) (*http.Response, error) {
	req, err := c.newRequest(http.MethodGet, path, params)
	if err != nil {
		return nil, err
	}

	startTime := time.Now()
	resp, err := c.hc.Do(req.WithContext(ctx))
	if err != nil {
		return nil, err
	}
	defer func() {
		if resp.Body != nil {
			_ = resp.Body.Close()
		}
	}()

	err, ok := c.checkResponse(req, resp, startTime)
	if ok {
		return resp, err
	}

	if e, ok := errorFromResponse(req, resp, "Checkpoint"); ok {
		return nil, e
	}
	if err = decodeResponseAsJSON(resp, resp.Body, output); err != nil {
		return nil, err
	}
	return resp, err
}

func (c *Client) newRequest(method string, path string, params url.Values) (*http.Request, error) {
	url := c.formatURL(path, params)

	req, err := http.NewRequest(method, url, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Accept", "application/json")
	if k := c.session; k != "" {
		req.Header.Set("Cookie", "checkpoint.session="+k)
	}

	return req, nil
}

func (c *Client) formatURL(path string, params url.Values) string {
	result := url.URL{
		Scheme: c.scheme,
		Host:   c.host,
		Path:   "/api/checkpoint/v1" + path,
	}
	if params != nil {
		result.RawQuery = params.Encode()
	}
	return result.String()
}

func (c *Client) checkResponse(
	req *http.Request,
	resp *http.Response,
	startTime time.Time) (error, bool) {
	c.logger.Infow(req.Method,
		"url", req.URL.String(),
		"time", time.Since(startTime).Seconds(),
		"status", resp.StatusCode)
	return errorFromResponse(req, resp, "Checkpoint")
}
