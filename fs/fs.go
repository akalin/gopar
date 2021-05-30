package fs

// FS is the interface used by the par1 and par2 packages to the
// filesystem. Most code uses DefaultFS, but tests may use other
// implementations.
type FS interface {
	// ReadFile should behave like ioutil.ReadFile.
	ReadFile(path string) ([]byte, error)
	// FindWithPrefixAndSuffix should behave like calling
	// filepath.Glob with prefix + "*" + suffix.
	FindWithPrefixAndSuffix(prefix, suffix string) ([]string, error)
	// WriteFile should behave like ioutil.WriteFile.
	WriteFile(path string, data []byte) error
}
