package utils

import "strings"

func IsAttachmentsFieldNotFound(err error) bool {
	if err == nil {
		return false
	}
	errStr := err.Error()
	return strings.Contains(errStr, "Cannot query field") && strings.Contains(errStr, "attachments")
}
