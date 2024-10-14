// This file contains utility functions for testing
package test

import (
	"fmt"
	"os"
	"testing"

	"kcl-lang.io/kpm/pkg/settings"
)

// acquireGlobalLock acquires the global lock for the package cache.
func acquireGlobalLock() error {
	err := settings.GetSettings().AcquirePackageCacheLock(os.Stdout)
	if err != nil {
		return err
	}
	return nil
}

// releaseGlobalLock releases the global lock for the package cache.
func releaseGlobalLock() error {
	err := settings.GetSettings().ReleasePackageCacheLock()
	if err != nil {
		return err
	}
	return nil
}

// RunTestWithGlobalLock runs a test with the global lock acquired.
func RunTestWithGlobalLock(t *testing.T, name string, testFunc func(t *testing.T)) {
	t.Run(name, func(t *testing.T) {
		err := acquireGlobalLock()
		if err != nil {
			t.Errorf("Error acquiring lock: %v", err)
		}

		defer func() {
			err := releaseGlobalLock()
			if err != nil {
				t.Errorf("Error releasing lock: %v", err)
			}
		}()

		testFunc(t)
		fmt.Printf("%s completed\n", name)
	})
}
