package fw

import (
	"context"
	"flag"
	"fmt"
	"io/fs"
	"log"
	"net/http"
	"os"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/joho/godotenv"
	"github.com/sjc5/hwy"
	"github.com/sjc5/kiruna"
	"github.com/sjc5/kit/pkg/envutil"
	"github.com/sjc5/kit/pkg/executil"
	"github.com/sjc5/kit/pkg/fsutil"
	"github.com/sjc5/kit/pkg/grace"
	"github.com/sjc5/kit/pkg/lazyget"
	"github.com/sjc5/kit/pkg/response"
	"github.com/sjc5/kit/pkg/theme"
	"github.com/sjc5/kit/pkg/tsgen"

	esbuild "github.com/evanw/esbuild/pkg/api"
	chimiddleware "github.com/go-chi/chi/v5/middleware"
)

// __TODO for opengraph checker, do we need a "kiruna.GetOriginalPublicURL"?

// AdHocTypes: []hwy.AdHocType{
// 	{Struct: lime.AuthPayload{}},
// 	{Struct: lime.DBSession[auth.ExtraUserData]{}, TSTypeName: "TypedDBSession"},
// },

type (
	// Instance
	AdHocTypes           = []hwy.AdHocType
	GenerateExtraTSCode  = func(statements *tsgen.Statements)
	DataFuncs            = hwy.DataFuncs // __TODO should this be a ptr
	FilesToVendor        = [][2]string
	RobotsTxt            = string
	IsOpenGraphImage     = func(urlPath string) bool
	RootID               = string
	DistFS               fs.FS
	GetDefaultHeadBlocks = func(r *http.Request) ([]hwy.HeadBlock, error)

	// Other
	// DevConfig = kiruna.DevConfig
	HeadBlock = hwy.HeadBlock
	Kiruna    = kiruna.Kiruna
)

type InstanceOptions[AHD any] struct {
	AdHocTypes             AdHocTypes
	GenerateExtraTSCode    GenerateExtraTSCode
	DataFuncs              *DataFuncs
	FilesToVendor          FilesToVendor
	RobotsTxt              RobotsTxt
	IsOpenGraphImage       IsOpenGraphImage
	GetAdHocDataForContext func(r *http.Request) (AHD, error)
	GetDefaultHeadBlocks   GetDefaultHeadBlocks
	RootID                 RootID
	Kiruna                 *kiruna.Kiruna
}

// Do not create a new instance of this struct directly. Use NewInstance instead,
// otherwise it will not be appropriately initialized.
type Instance[AHD any, SE any, CEE any] struct {
	*InstanceOptions[AHD]

	hwyBuildOptions *hwy.BuildOptions
	GetEnv          func() *Env[SE, CEE]
}

func New[AHD any, SE any, CEE any](opts *InstanceOptions[AHD]) *Instance[AHD, SE, CEE] {
	instance := &Instance[AHD, SE, CEE]{InstanceOptions: opts}

	instance.GetEnv = lazyget.New(func() *Env[SE, CEE] {
		if err := godotenv.Load(); err != nil {
			panic(err)
		}

		env := &Env[SE, CEE]{}

		env.Meta.Mode = envutil.GetStr(EnvModeKey, EnvModeValueProd)

		if env.Meta.Mode != EnvModeValueDev && env.Meta.Mode != EnvModeValueProd {
			panic(fmt.Sprintf("env: invalid MODE value: %s", env.Meta.Mode))
		}

		env.Meta.IsDev = env.Meta.Mode == EnvModeValueDev
		env.Meta.IsProd = env.Meta.Mode == EnvModeValueProd
		env.Meta.Port = envutil.GetInt("PORT", kiruna.MustGetPort())
		env.Meta.DSN = envutil.GetStr("DSN", "")
		env.Meta.CurrentCookieSecret = envutil.GetStr("CURRENT_COOKIE_SECRET", "")
		env.Meta.PreviousCookieSecret = envutil.GetStr("PREVIOUS_COOKIE_SECRET", "")
		env.Meta.SecretDecoyPepper = envutil.GetStr("SECRET_DECOY_PEPPER", "")
		env.ClientExposed.Meta.Mode = env.Meta.Mode

		if env.Meta.IsProd {
			if env.Meta.DSN == "" {
				panic(toProdRequiredErrMsg("DSN"))
			}
			if env.Meta.CurrentCookieSecret == "" {
				panic(toProdRequiredErrMsg("CURRENT_COOKIE_SECRET"))
			}
			if env.Meta.PreviousCookieSecret == "" {
				panic(toProdRequiredErrMsg("PREVIOUS_COOKIE_SECRET"))
			}
			if env.Meta.SecretDecoyPepper == "" {
				panic(toProdRequiredErrMsg("SECRET_DECOY_PEPPER"))
			}
		}

		return env
	})

	return instance
}

func toProdRequiredErrMsg(name string) string {
	return fmt.Sprintf("env: %s must be set in production mode", name)
}

func NewKiruna(distFS fs.FS) *kiruna.Kiruna {
	return kiruna.New(&kiruna.Config{
		DistFS:           distFS,
		MainAppEntry:     "cmd/app/main.go",
		PrivateStaticDir: "static/private",
		PublicStaticDir:  "static/public",
		StylesDir:        "styles",
		DistDir:          "dist",
	})
}

func (fw *Instance[AHD, SE, CEE]) Gen() {
	tsGenOptions := &hwy.TSGenOptions{
		DataFuncs:   fw.DataFuncs,
		OutPath:     "frontend/hwy_tsgen.ts",
		AdHocTypes:  fw.AdHocTypes,
		ExtraTSCode: fw.getExtraTSCode(),
	}

	if err := hwy.GenerateTypeScript(tsGenOptions); err != nil {
		panic(err)
	}
}

// Why is this pattern necessary? Because TS is generated
// from within the dev server, and the dev server is
// instantiated only once, not every time you save a .go
// file. So, in order for the types to actually be
// re-evaluated, we need to run this file as a fresh
// script every time we save a .go file.
func (fw *Instance[AHD, SE, CEE]) tsGenCmdRunner() error {
	return executil.MakeCmdRunner("go", "run", "./cmd/tasks", "-gen")()
}

func (fw *Instance[AHD, SE, CEE]) vendorFiles() error {
	for _, c := range fw.FilesToVendor {
		if err := fsutil.CopyFile(c[0], c[1]); err != nil {
			return err
		}
	}

	return nil
}

func (fw *Instance[AHD, SE, CEE]) mustMultiBuild() {
	if err := fw.vendorFiles(); err != nil {
		panic(err)
	}

	// Must run once at beginning so that it exists when MustGenTS is called
	if err := fw.Kiruna.BuildWithoutCompilingGo(); err != nil {
		panic(err)
	}

	fw.tsGenCmdRunner()

	if err := fw.buildHwy(); err != nil {
		panic(err)
	}
}

func (fw *Instance[AHD, SE, CEE]) buildHwy() error {
	if fw.hwyBuildOptions == nil {
		fw.hwyBuildOptions = &hwy.BuildOptions{
			// inputs
			IsDev:       fw.GetEnv().Meta.IsDev,
			ClientEntry: "frontend/entry.tsx",
			PagesSrcDir: "frontend/routes",
			DataFuncs:   fw.DataFuncs,

			// __PREACT_SPECIFIC
			UsePreactCompat: true,

			// outputs
			StaticPublicOutDir:  "static/public/prehashed",
			StaticPrivateOutDir: "static/private",

			// esbuild plugins
			ESBuildPlugins: []esbuild.Plugin{
				hwy.CreateCSSURLFuncResolverPlugin(fw.Kiruna.GetPublicURL),
				hwy.CreatePublicURLResolverPlugin("getPublicURLBuildtime", fw.Kiruna.GetPublicURL),
			},
		}
	}

	return hwy.Build(fw.hwyBuildOptions)
}

func (fw *Instance[AHD, SE, CEE]) Dev() {
	if err := SetDevMode(); err != nil {
		panic(err)
	}

	fw.mustMultiBuild()

	fw.Kiruna.MustStartDev(&kiruna.DevConfig{
		HealthcheckEndpoint: "/healthz",
		IgnorePatterns: kiruna.IgnorePatterns{
			Dirs: []string{
				"__*",
				"static/public/prehashed",
			},
			Files: []string{
				"**/*.d.ts",
				"frontend/hwy_tsgen.ts",
				"static/private/" + hwy.HwyPathsFileName,
			},
		},
		WatchedFiles: kiruna.WatchedFiles{
			{
				Pattern: "**/*.go",
				OnChangeCallbacks: []kiruna.OnChange{{
					Func:     func(_ string) error { return fw.tsGenCmdRunner() },
					Strategy: kiruna.OnChangeStrategyConcurrent,
				}},
			},
			{
				Pattern: "**/*.{ts,tsx}",
				OnChangeCallbacks: []kiruna.OnChange{{
					Func: func(_ string) error { return fw.buildHwy() },
				}},
				RestartApp: true,
			},
			{
				Pattern: "frontend/**/*.css",
				OnChangeCallbacks: []kiruna.OnChange{{
					Func: func(_ string) error { return fw.buildHwy() },
				}},
				RestartApp:                     true,
				RunClientDefinedRevalidateFunc: true,
				SkipRebuildingNotification:     true,
			},
			{
				Pattern:    "**/*.go.html",
				RestartApp: true,
			},
			// __TODO
			// {
			// 	Pattern:    "static/private/markdown/*.md",
			// 	RestartApp: true,
			// },
		},
	})
}

const (
	EnvModeKey       = "MODE"
	EnvModeValueProd = "production"
	EnvModeValueDev  = "development"
)

// SetDevMode sets the MODE environment variable to "development".
func SetDevMode() error {
	return os.Setenv(EnvModeKey, EnvModeValueDev)
}

func (fw *Instance[AHD, SE, CEE]) initRouter() *chi.Mux {
	r := chi.NewRouter()

	r.Use(
		chimiddleware.RequestID,
		chimiddleware.Logger,
		chimiddleware.Recoverer,
		chimiddleware.Heartbeat("/healthz"),
		fw.openGraphCrossOriginFixer,
		// fakeDelayMiddleware,
		// auth.AuthManager.ToSessionMiddleware(),
		// auth.AuthManager.ToCSRFMiddleware(),
	)

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

	return r
}

func (fw *Instance[AHD, SE, CEE]) openGraphCrossOriginFixer(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if fw.IsOpenGraphImage(r.URL.Path) {
			w.Header().Set("Access-Control-Allow-Origin", "*")
			w.Header().Set("Cross-Origin-Resource-Policy", "cross-origin")
		}
		next.ServeHTTP(w, r)
	})
}

func GetAdHocDataFromContext[T any](r *http.Request) T {
	return hwy.GetAdHocDataFromContext[T](r)
}

func (fw *Instance[AHD, SE, CEE]) adHocDataMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		val, err := fw.GetAdHocDataForContext(r)
		if err != nil {
			fmt.Println("Error getting ad hoc data for context:", err)
			http.Error(w, "Internal server error", http.StatusInternalServerError)
			return
		}
		next.ServeHTTP(w, r.WithContext(hwy.GetAdHocDataContextWithValue(r, val)))
	})
}

func (fw *Instance[AHD, SE, CEE]) getHwy() *hwy.Hwy {
	privateFS, err := fw.Kiruna.GetPrivateFS()
	if err != nil {
		panic(fmt.Sprintf("Error loading private FS: %v", err))
	}

	hwyInstance := hwy.Hwy{
		FS:                   privateFS,
		PublicURLResolver:    fw.Kiruna.GetPublicURL,
		RootTemplateLocation: "index.go.html",
		Loaders:              fw.DataFuncs.Loaders,
		QueryActions:         fw.DataFuncs.QueryActions,
		MutationActions:      fw.DataFuncs.MutationActions,
		GetDefaultHeadBlocks: fw.GetDefaultHeadBlocks,
		GetRootTemplateData: func(r *http.Request) (map[string]any, error) {
			return map[string]any{
				"Kiruna":                      fw.Kiruna,
				"HTMLClass":                   theme.GetThemeData(r).HTMLClass,
				"RootID":                      fw.RootID,
				"SystemThemeScript":           theme.SystemThemeScript,
				"SystemThemeScriptSha256Hash": theme.SystemThemeScriptSha256Hash,
			}, nil
		},
	}

	err = hwyInstance.Init()
	if err != nil {
		errMsg := fmt.Sprintf("Error initializing Hwy: %v", err)
		if fw.GetEnv().Meta.IsDev {
			fmt.Println(errMsg)
		} else {
			panic(errMsg)
		}
	} else {
		fmt.Println("Hwy initialized")
	}

	return &hwyInstance
}

func (fw *Instance[AHD, SE, CEE]) getExtraTSCode() string {
	a := tsgen.Statements{}

	a.Serialize("export const ROOT_ID", fw.RootID)
	keys, err := fw.Kiruna.GetPublicFileMapKeysBuildtime([]string{"hwy_chunk__", "hwy_entry__"})
	if err != nil {
		panic(err)
	}
	a.Serialize("const PUBLIC_FILE_MAP_KEYS", keys)
	a.Raw("export type PublicFileMapKey", "typeof PUBLIC_FILE_MAP_KEYS[number]")

	if fw.GenerateExtraTSCode != nil {
		fw.GenerateExtraTSCode(&a)
	}

	return a.String()
}

func (fw *Instance[AHD, SE, CEE]) Build() {
	fw.mustMultiBuild()

	if err := fw.Kiruna.Build(); err != nil {
		panic(err)
	}
}

func (fw *Instance[AHD, SE, CEE]) App() {
	port := fw.GetEnv().Meta.Port
	r := fw.initRouter()

	server := &http.Server{
		Addr:                         fmt.Sprintf(":%d", port),
		Handler:                      http.TimeoutHandler(r, 60*time.Second, "Request timed out"),
		ReadTimeout:                  15 * time.Second,
		WriteTimeout:                 30 * time.Second,
		IdleTimeout:                  60 * time.Second,
		ReadHeaderTimeout:            10 * time.Second,
		MaxHeaderBytes:               1 << 20, // 1 MB
		DisableGeneralOptionsHandler: true,
		ErrorLog:                     log.New(os.Stderr, "HTTP: ", log.Ldate|log.Ltime|log.Lshortfile),
	}

	grace.Orchestrate(grace.OrchestrateOptions{
		StartupCallback: func() error {
			log.Println("Starting server on: http://localhost:", port)

			if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
				log.Fatalf("Server listen and serve error: %v\n", err)
			}

			return nil
		},
		ShutdownCallback: func(shutdownCtx context.Context) error {
			log.Println("Shutting down server on: http://localhost:", port)

			if err := server.Shutdown(shutdownCtx); err != nil {
				log.Fatalf("Server shutdown error: %v\n", err)
			}

			return nil
		},
	})
}

func (fw *Instance[AHD, SE, CEE]) Tasks() {
	devFlag := flag.Bool("dev", false, "Run Dev function")
	buildFlag := flag.Bool("build", false, "Run Build function")
	genFlag := flag.Bool("gen", false, "Run Gen function")

	flag.Parse()

	// Count how many flags are set to true
	flagCount := 0
	if *devFlag {
		flagCount++
	}
	if *buildFlag {
		flagCount++
	}
	if *genFlag {
		flagCount++
	}

	// Panic if no flags or multiple flags are set
	if flagCount == 0 {
		panic("No command flag specified. Use one of: -main, -dev, -build, -gen")
	}
	if flagCount > 1 {
		panic("Only one command flag can be specified at a time")
	}

	// Run the appropriate function based on the flag
	switch {
	case *devFlag:
		fw.Dev()
	case *buildFlag:
		fw.Build()
	case *genFlag:
		fw.Gen()
	}
}

type EnvMetaServer struct {
	Mode                 string
	IsDev                bool
	IsProd               bool
	Port                 int
	DSN                  string
	CurrentCookieSecret  string
	PreviousCookieSecret string
	SecretDecoyPepper    string
}

type EnvMetaClientExposed struct {
	Mode string `json:"mode"`
}

type Env[S any, CE any] struct {
	Meta          EnvMetaServer
	User          S
	ClientExposed struct {
		Meta EnvMetaClientExposed `json:"meta"`
		User CE
	}
}
