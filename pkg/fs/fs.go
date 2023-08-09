package fs

import (
	"errors"
	"os"
)

func CreateIfNotExist(filepath string) (*os.File, error) {
	var (
		err  error
		file *os.File
	)

	if _, err = os.Stat(filepath); err == nil {
		return nil, errors.New("file exists")
	}

	if file, err = os.Create(filepath); err != nil {
		return nil, err
	}

	return file, nil
}
