package main

import (
	"github.com/TogetherForStudy/jxust-yqlx-server/internal/bootstrap"
	_ "github.com/TogetherForStudy/jxust-yqlx-server/internal/perf"
)

func main() {
	app := bootstrap.New()
	app.Run()
}
