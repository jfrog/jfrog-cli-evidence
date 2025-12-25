package utils

func IsHttpStatusSuccessful(statusCode int) bool {
	return statusCode >= 200 && statusCode <= 299
}
