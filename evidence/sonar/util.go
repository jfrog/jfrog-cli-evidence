package sonar

import (
	"github.com/jfrog/jfrog-client-go/utils/io/fileutils"
	"github.com/jfrog/jfrog-client-go/utils/log"
)

func fileExists(path string) bool {
	if path == "" {
		return false
	}
	exists, err := fileutils.IsFileExists(path, false)
	if err != nil {
		log.Debug("Error while checking if file exists", path, err)
	}
	return err == nil && exists
}
