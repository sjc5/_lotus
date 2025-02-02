package glue

import (
	esbuild "github.com/evanw/esbuild/pkg/api"
	"github.com/sjc5/hwy"
	"github.com/sjc5/kit/pkg/fsutil"
)

func (fw *Instance[AHD, SE, CEE]) Build() {
	fw.mustMultiBuild()

	if err := fw.Kiruna.Build(); err != nil {
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

func (fw *Instance[AHD, SE, CEE]) vendorFiles() error {
	for _, c := range fw.FilesToVendor {
		if err := fsutil.CopyFile(c[0], c[1]); err != nil {
			return err
		}
	}

	return nil
}
