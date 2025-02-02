package glue

import (
	"io/fs"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/sjc5/hwy"
	"github.com/sjc5/kiruna"
)

// __TODO for opengraph checker, do we need a "kiruna.GetOriginalPublicURL"?
// __TODO make much more configurable via pass-throughs / defaults
// __TODO add in auth middleware
// __TODO add frontend companion
// __TODO make hwy usage optional?

type (
	AdHocTypes           = []AdHocType
	GenerateExtraTSCode  func(statements *Statements)
	DataFuncs            = hwy.DataFuncs
	FilesToVendor        = [][2]string
	RobotsTxt            = string
	IsOpenGraphImage     func(urlPath string) bool
	RootID               = string
	DistFS               fs.FS
	GetDefaultHeadBlocks func(r *http.Request) ([]HeadBlock, error)
)

type InstanceOptions[AHD any, SE any, CEE any] struct {
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
	GeneralMiddlewares     Middlewares
	ModifyRouter           func(r *chi.Mux)
	GetEnv                 GetEnv[SE, CEE]
}

// Do not create a new instance of this struct directly. Use NewInstance instead,
// otherwise it will not be appropriately initialized.
type Instance[AHD any, SE any, CEE any] struct {
	*InstanceOptions[AHD, SE, CEE]

	hwyBuildOptions *hwy.BuildOptions
	GetEnv          func() *Env[SE, CEE]
}

func New[AHD any, SE any, CEE any](opts *InstanceOptions[AHD, SE, CEE]) *Instance[AHD, SE, CEE] {
	instance := &Instance[AHD, SE, CEE]{InstanceOptions: opts}

	if instance.GetEnv == nil {
		instance.GetEnv = MakeGetEnv[SE, CEE]()
	}

	return instance
}
