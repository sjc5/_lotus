package glue

import (
	"fmt"
	"os"

	"github.com/joho/godotenv"
	"github.com/sjc5/kiruna"
	"github.com/sjc5/kit/pkg/envutil"
	"github.com/sjc5/kit/pkg/lazyget"
)

const (
	EnvModeKey       = "MODE"
	EnvModeValueProd = "production"
	EnvModeValueDev  = "development"
)

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

type GetEnv[SE any, CEE any] func() *Env[SE, CEE]

func MakeGetEnv[SE any, CEE any]() GetEnv[SE, CEE] {
	return lazyget.New(func() *Env[SE, CEE] {
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
}

// SetDevMode sets the MODE environment variable to "development".
func envSetDevMode() error {
	return os.Setenv(EnvModeKey, EnvModeValueDev)
}

func toProdRequiredErrMsg(name string) string {
	return fmt.Sprintf("env: %s must be set in production mode", name)
}
