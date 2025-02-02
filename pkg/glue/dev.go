package glue

import (
	"github.com/sjc5/hwy"
	"github.com/sjc5/kiruna"
)

func (fw *Instance[AHD, SE, CEE]) Dev() {
	if err := envSetDevMode(); err != nil {
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
