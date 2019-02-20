package utils

import (
	"errors"
	"fmt"
	"os"
	"os/user"
	"path"
)

func GetDefaultDBPath() string {
	return "./"
}

func CheckDirExists(path string) (bool, error) {
	info, err := os.Stat(path)
	if err == nil {
		return info.IsDir(), nil
	}

	return false, err
}

func CheckFileExists(file string) (bool, error) {
	info, err := os.Stat(file)
	if err == nil {
		if info.IsDir() {
			errMsg := fmt.Sprintf("'%s' is a directory", file)
			return false, errors.New(errMsg)
		}

		return true, nil
	}

	if os.IsNotExist(err) {
		return false, nil
	}

	return false, err
}

func CheckWriteable(dir string) error {
	_, err := os.Stat(dir)
	if err == nil {
		// dir exists, make sure we can write to it
		testfile := path.Join(dir, "test")
		fi, err := os.Create(testfile)
		if err != nil {
			if os.IsPermission(err) {
				return fmt.Errorf("%s is not writeable by the current user", dir)
			}
			return fmt.Errorf("unexpected error while checking writeablility of repo root: %s", err)
		}
		fi.Close()
		return os.Remove(testfile)
	}

	if os.IsNotExist(err) {
		// dir doesnt exist, check that we can create it
		return os.Mkdir(dir, 0775)
	}

	if os.IsPermission(err) {
		return fmt.Errorf("cannot write to %s, incorrect permissions", err)
	}

	return err
}

func GetHomeDir() string {
	user, _ := user.Current()
	return user.HomeDir
}
