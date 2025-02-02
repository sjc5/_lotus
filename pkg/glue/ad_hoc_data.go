package glue

import (
	"fmt"
	"net/http"

	"github.com/sjc5/hwy"
)

func GetAdHocDataFromContext[T any](r *http.Request) T {
	return hwy.GetAdHocDataFromContext[T](r)
}

func (fw *Instance[AHD, SE, CEE]) adHocDataMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if fw.GetAdHocDataForContext == nil {
			next.ServeHTTP(w, r)
			return
		}

		val, err := fw.GetAdHocDataForContext(r)
		if err != nil {
			fmt.Println("Error getting ad hoc data for context:", err)
			http.Error(w, "Internal server error", http.StatusInternalServerError)
			return
		}

		next.ServeHTTP(w, r.WithContext(hwy.GetAdHocDataContextWithValue(r, val)))
	})
}
