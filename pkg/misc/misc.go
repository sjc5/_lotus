package misc

import (
	"net/http"
	"strings"
	"time"
)

func ToFakeDelayMiddleware(delay time.Duration) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if !strings.HasPrefix(r.URL.Path, "/public/") {
				time.Sleep(delay)
			}
			next.ServeHTTP(w, r)
		})
	}
}

func MakeEmojiDataURL(emojiStr string) string {
	sb := strings.Builder{}
	sb.WriteString("data:image/svg+xml,")
	sb.WriteString("<svg xmlns='http://www.w3.org/2000/svg' width='48' height='48' viewBox='0 0 16 16'>")
	sb.WriteString("<text x='0' y='14'>")
	sb.WriteString(emojiStr)
	sb.WriteString("</text>")
	sb.WriteString("</svg>")
	return sb.String()
}
