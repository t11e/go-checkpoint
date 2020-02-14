package checkpoint

import (
	"fmt"
	"net/http"
	"time"
)

// SessionFromRequest finds the Checkpoint session from a request.
func SessionFromRequest(req *http.Request) (string, bool) {
	if session, ok := req.URL.Query()["session"]; ok {
		if len(session) == 1 && session[0] != "" {
			return session[0], true
		}
	}
	if s := req.Header.Get("x-checkpoint-session"); s != "" {
		return s, true
	}
	c, err := req.Cookie("checkpoint.session")
	if err == nil && c.Value != "" {
		return c.Value, true
	}
	return "", false
}

func AddResponseHeader(header http.Header, sessionKey string, expiry *time.Duration) {
	var exp time.Duration
	if expiry != nil {
		exp = *expiry
	} else {
		exp = defaultExpiry
	}
	header.Add("Set-Cookie", fmt.Sprintf("checkpoint.session=%s; expires=%s; HttpOnly", sessionKey,
		time.Now().Add(exp).UTC().Format(time.RFC822)))
}

var defaultExpiry = 24 * time.Hour * 365
