package glue

import (
	"fmt"
	"net/http"

	"github.com/sjc5/hwy"
	"github.com/sjc5/kit/pkg/theme"
)

type HeadBlock = hwy.HeadBlock

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
