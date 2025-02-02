package glue

import (
	"net/http"

	"github.com/go-chi/chi/v5"
	chimiddleware "github.com/go-chi/chi/v5/middleware"
	"github.com/sjc5/kit/pkg/response"
)

type Middleware = func(http.Handler) http.Handler
type Middlewares = []Middleware

func (fw *Instance[AHD, SE, CEE]) initRouter() *chi.Mux {
	r := chi.NewRouter()

	mw := []func(http.Handler) http.Handler{
		chimiddleware.RequestID,
		chimiddleware.Logger,
		chimiddleware.Recoverer,
		chimiddleware.Heartbeat("/healthz"),
		fw.openGraphCrossOriginFixer,
	}

	if fw.GeneralMiddlewares != nil {
		mw = append(mw, fw.GeneralMiddlewares...)
	}

	r.Use(mw...)

	r.HandleFunc("/robots.txt", func(w http.ResponseWriter, r *http.Request) {
		response.New(w).Text(fw.RobotsTxt)
	})

	r.Handle("/public/*", fw.Kiruna.MustGetServeStaticHandler("/public/", true))

	r.HandleFunc("/favicon.ico", func(w http.ResponseWriter, r *http.Request) {
		faviconDotIcoURL := fw.Kiruna.GetPublicURL("favicon.ico")
		if faviconDotIcoURL == "/public/favicon.ico" {
			response.New(w).NotFound()
			return
		}
		http.Redirect(w, r, faviconDotIcoURL, http.StatusFound)
	})

	r.Group(func(r chi.Router) {
		r.Use(fw.adHocDataMiddleware)
		r.Handle("/*", fw.getHwy().GetRootHandler())
	})

	if fw.ModifyRouter != nil {
		fw.ModifyRouter(r)
	}

	return r
}

func (fw *Instance[AHD, SE, CEE]) openGraphCrossOriginFixer(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if fw.IsOpenGraphImage != nil && fw.IsOpenGraphImage(r.URL.Path) {
			w.Header().Set("Access-Control-Allow-Origin", "*")
			w.Header().Set("Cross-Origin-Resource-Policy", "cross-origin")
		}
		next.ServeHTTP(w, r)
	})
}
