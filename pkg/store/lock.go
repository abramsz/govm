package store

import (
	"fmt"
	"os"
	"path/filepath"
	"time"
)

const lockFileName = ".lock"

// retryInterval and retryMax control how long Lock() waits before giving up.
// Exported for testing (can be shortened).
var (
	retryInterval = 200 * time.Millisecond
	retryMax      = 25 // 25 × 200ms = 5 seconds total
)

// Lock acquires an exclusive file lock on the govm store.
// Uses O_CREATE|O_EXCL, which is atomic on local filesystems (NTFS, ext4, etc.)
// and works cross-platform (Linux, macOS, Windows).
func Lock() error {
	path := filepath.Join(Home(), lockFileName)
	for range retryMax {
		f, err := os.OpenFile(path, os.O_CREATE|os.O_EXCL, 0o644)
		if err == nil {
			fmt.Fprintf(f, "%d\n", os.Getpid())
			f.Close()
			return nil
		}
		if !os.IsExist(err) {
			return fmt.Errorf("acquire lock: %w", err)
		}
		time.Sleep(retryInterval)
	}
	return fmt.Errorf("govm store is locked — another govm process is running in %s", Home())
}

// Unlock releases the lock acquired by Lock.
func Unlock() {
	_ = os.Remove(filepath.Join(Home(), lockFileName))
}
