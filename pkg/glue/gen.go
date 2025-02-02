package glue

import (
	"github.com/sjc5/hwy"
	"github.com/sjc5/kit/pkg/executil"
	"github.com/sjc5/kit/pkg/tsgen"
)

type (
	Statements = tsgen.Statements
	AdHocType  = hwy.AdHocType
)

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
