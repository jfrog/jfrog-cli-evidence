package cli

import (
	"github.com/jfrog/jfrog-cli-core/v2/plugins/components"
)

func GetJfrogCliEvidenceApp() components.App {
	app := components.CreateEmbeddedApp(
		"evidence",
		[]components.Command{},
		components.Namespace{
			Name:        "evd",
			Description: "Evidence commands.",
			Commands:    GetCommands(),
			Category:    "Command Namespaces",
		},
	)
	return app
}

func GetStandaloneEvidenceApp() components.App {
	app := components.CreateApp(
		"evd",
		"v1.0.0",
		"Evidence commands.",
		GetCommands(),
	)
	return app
}
