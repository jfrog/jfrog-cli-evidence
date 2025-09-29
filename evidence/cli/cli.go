package cli

import (
	"github.com/jfrog/jfrog-cli-core/v2/plugins/components"
	"github.com/jfrog/jfrog-cli-evidence/evidence/cli/command"
)

func GetJfrogCliEvidenceApp() components.App {
	app := components.CreateEmbeddedApp(
		"evidence",
		[]components.Command{},
		components.Namespace{
			Name:        "evd",
			Description: "Evidence command.",
			Commands:    command.GetCommands(),
			Category:    "Command Namespaces",
		},
	)
	return app
}

func GetStandaloneEvidenceApp() components.App {
	app := components.CreateApp(
		"evd",
		"v1.0.0",
		"Evidence command.",
		command.GetCommands(),
	)
	return app
}
