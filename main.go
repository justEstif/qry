package main

import (
	"github.com/justestif/qry/cmd"

	// Register built-in adapters
	_ "github.com/justestif/qry/adapters/braveapi"
	_ "github.com/justestif/qry/adapters/bravescrape"
	_ "github.com/justestif/qry/adapters/ddgscrape"
	_ "github.com/justestif/qry/adapters/exa"
	_ "github.com/justestif/qry/adapters/github"
	_ "github.com/justestif/qry/adapters/mock"
	_ "github.com/justestif/qry/adapters/searx"
	_ "github.com/justestif/qry/adapters/stackoverflow"
	_ "github.com/justestif/qry/adapters/wikipedia"
)

// version is set at build time via -ldflags="-X main.version=<tag>"
var version = "dev"

func main() {
	cmd.Execute(version)
}
