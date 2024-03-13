package cryptox

func ReadKey(fileContent []byte) (*SSLibKey, error) {
	slibKey, err := LoadKey(fileContent)
	if err != nil {
		return nil, err
	}
	if slibKey.KeyVal.Private != "" {
		return slibKey, nil
	}

	return nil, nil
}
