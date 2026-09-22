// Package jev は Jev への HTTP 呼び出しを持つ。
package jev

import (
	"net/http"
	"os"
	"time"
)

// Client は計画を作るときの Jev への送信を持つ。
type Client struct {
	HTTP *http.Client
	URL  string
	Key  string
}

// NewFromEnv は URL と鍵を環境変数から読む。呼び出し先が無いときは送らない。
func NewFromEnv() *Client {
	return &Client{
		HTTP: &http.Client{Timeout: 60 * time.Second},
		URL:  os.Getenv("ASUYARU_JEV_URL"),
		Key:  os.Getenv("ASUYARU_JEV_KEY"),
	}
}

// Configured は URL と鍵が両方あるときに真である。
func (c *Client) Configured() bool {
	return c != nil && c.URL != "" && c.Key != ""
}
