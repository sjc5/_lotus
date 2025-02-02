package glue

import (
	"io/fs"

	"github.com/sjc5/kiruna"
)

type Kiruna = kiruna.Kiruna

func NewKiruna(distFS fs.FS) *kiruna.Kiruna {
	return kiruna.New(&kiruna.Config{
		DistFS:           distFS,
		MainAppEntry:     "cmd/app/main.go",
		DistDir:          "dist",
		PrivateStaticDir: "static/private",
		PublicStaticDir:  "static/public",
		StylesDir:        "styles",
	})
}
