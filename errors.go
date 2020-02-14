package checkpoint

import (
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"mime"
	"net/http"

	"github.com/pkg/errors"
)

func isStatus(err error, status int) bool {
	e, ok := err.(*Error)
	return ok && e.Resp.StatusCode == status
}

type Error struct {
	Req     *http.Request
	Resp    *http.Response
	Body    string
	Message string
}

func (e *Error) Error() string {
	limit := 500
	body := e.Body
	if len(body) > limit {
		body = body[0:limit] + fmt.Sprintf("... [%d more bytes]", limit-len(body))
	}
	return fmt.Sprintf("%s: Request [%s %s] failed with status %d (%s): %s",
		e.Message, e.Req.Method, e.Req.URL, e.Resp.StatusCode, e.Resp.Status, e.Body)
}

func errorFromResponse(req *http.Request, resp *http.Response, message string, format ...interface{}) (error, bool) {
	if resp.StatusCode >= 200 && resp.StatusCode <= 299 {
		return nil, false
	}

	defer func() {
		if resp.Body != nil {
			_ = resp.Body.Close()
		}
	}()

	var body string
	if resp.Body != nil {
		b, err := ioutil.ReadAll(resp.Body)
		if err != nil {
			body = fmt.Sprintf("[error reading response body: %s]", err)
		} else if len(b) > 0 {
			body = string(b)
		} else {
			body = "[no data in response]"
		}
	}

	return &Error{
		Resp:    resp,
		Req:     req,
		Body:    body,
		Message: fmt.Sprintf(message, format...),
	}, true
}

// decodeResponseAsJSON decodes a HTTP client response as JSON.
func decodeResponseAsJSON(
	resp *http.Response,
	body io.Reader,
	out interface{}) error {
	if resp.ContentLength == 0 {
		// We treat this is as a non-error
		return nil
	}

	contentType := resp.Header.Get("Content-Type")
	if contentType == "" {
		return errors.New("expected response to be JSON, received bytes")
	}

	if mediaType, _, err := mime.ParseMediaType(contentType); err != nil {
		return fmt.Errorf("invalid content type %q: %w", contentType, err)
	} else if mediaType != "application/json" {
		return fmt.Errorf("expected response to be JSON, got %q", mediaType)
	}

	b, err := ioutil.ReadAll(body)
	if err != nil {
		return fmt.Errorf("could not read entire response: %w", err)
	}

	if err := json.Unmarshal(b, out); err != nil {
		return fmt.Errorf("could not decode response JSON: %w", err)
	}

	return nil
}
