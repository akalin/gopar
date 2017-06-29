package par1

import "io/ioutil"

type fileIO interface {
	ReadFile(path string) ([]byte, error)
	WriteFile(path string, data []byte) error
}

type defaultFileIO struct{}

func (io defaultFileIO) ReadFile(path string) ([]byte, error) {
	return ioutil.ReadFile(path)
}

func (io defaultFileIO) WriteFile(path string, data []byte) error {
	return ioutil.WriteFile(path, data, 0600)
}
