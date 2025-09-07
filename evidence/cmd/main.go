package main

import (
	"github.com/jfrog/jfrog-cli-core/v2/plugins"
	"github.com/jfrog/jfrog-cli-evidence/evidence/cli"
)

func main() {
	plugins.PluginMain(cli.GetStandaloneEvidenceApp())
}
