package providers

import (
	"net/http"
	"net/http/cookiejar"
	"time"
)

func newWebClient() *http.Client {
	cookieJar, _ := cookiejar.New(nil)
	return &http.Client{Timeout: time.Second * 2, Jar: cookieJar}
}
