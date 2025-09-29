package test

import "github.com/jfrog/jfrog-cli-core/v2/plugins/components"

func SetDefaultValue(flag string, defaultValue string) components.Flag {
	f := components.NewStringFlag(flag, flag)
	f.DefaultValue = defaultValue
	return f
}
