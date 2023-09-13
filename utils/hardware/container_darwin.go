package hardware

import (
	"github.com/cockroachdb/errors"
)

// inContainer checks if the service is running inside a container
// It should be always false while under windows.
func inContainer() (bool, error) {
	return false, nil
}

// getContainerMemLimit returns memory limit and error
func getContainerMemLimit() (uint64, error) {
	return 0, errors.New("Not supported")
}

// getContainerMemUsed returns memory usage and error
func getContainerMemUsed() (uint64, error) {
	return 0, errors.New("Not supported")
}
