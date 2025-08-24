package database

import (
	"github.com/gofrs/flock"
)

type LockChecker interface {
	Locked() (bool, error)
}

type VFSLockChecker struct {
	path string
}

func (checker *VFSLockChecker) Locked() (bool, error) {
	fileLock := flock.New(checker.path)
	
	// Essayer de prendre le lock avec un timeout immédiat
	locked, err := fileLock.TryLock()
	if err != nil {
		return false, err
	}
	
	if locked {
		// On a réussi à prendre le lock, donc il n'était pas locked
		fileLock.Unlock()
		return false, nil
	}
	
	// On n'a pas réussi à prendre le lock, donc il est locked par quelqu'un d'autre
	return true, nil
}