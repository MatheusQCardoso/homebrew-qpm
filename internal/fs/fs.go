package fs

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
)

func EnsureDir(path string) error {
	return os.MkdirAll(path, 0o755)
}

func RemoveAll(path string) error {
	if path == "" || path == "/" {
		return fmt.Errorf("refusing to remove unsafe path %q", path)
	}
	return os.RemoveAll(path)
}

func FileExists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}

func ReadFile(path string) ([]byte, error) {
	return os.ReadFile(path)
}

func WriteFile(path string, data []byte) error {
	if err := EnsureDir(filepath.Dir(path)); err != nil {
		return err
	}
	return os.WriteFile(path, data, 0o644)
}

func CopyFile(dst, src string) error {
	sourceFile, err := os.Open(src)
	if err != nil {
		return err
	}
	defer sourceFile.Close()

	if err := EnsureDir(filepath.Dir(dst)); err != nil {
		return err
	}
	destinationFile, err := os.Create(dst)
	if err != nil {
		return err
	}
	defer func() { _ = destinationFile.Close() }()

	if _, err := io.Copy(destinationFile, sourceFile); err != nil {
		return err
	}
	return destinationFile.Close()
}

func CopyDir(dst, src string) error {
	sourceInfo, err := os.Stat(src)
	if err != nil {
		return err
	}
	if !sourceInfo.IsDir() {
		return fmt.Errorf("source is not a directory: %s", src)
	}

	if err := EnsureDir(dst); err != nil {
		return err
	}

	directoryEntries, err := os.ReadDir(src)
	if err != nil {
		return err
	}
	for _, entry := range directoryEntries {
		sourcePath := filepath.Join(src, entry.Name())
		destinationPath := filepath.Join(dst, entry.Name())

		if entry.IsDir() {
			if err := CopyDir(destinationPath, sourcePath); err != nil {
				return err
			}
			continue
		}

		if err := CopyFile(destinationPath, sourcePath); err != nil {
			return err
		}
	}

	return nil
}

func Symlink(dst, src string) error {
	if err := EnsureDir(filepath.Dir(dst)); err != nil {
		return err
	}
	_ = os.RemoveAll(dst)
	return os.Symlink(src, dst)
}
