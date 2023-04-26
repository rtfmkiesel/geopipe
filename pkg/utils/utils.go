package utils

import (
	"fmt"
	"os"
)

var (
	// If '-s' is supplied
	Silent bool
)

// Contains() will return true if a []string contains the specified string
func Contains(list []string, query string) bool {
	for _, item := range list {
		if item == query {
			return true
		}
	}

	return false
}

// CatchErr() will handle errors
func CatchErr(err error) {
	if err != nil && !Silent {
		fmt.Printf("ERROR: %s\n", err)
	}
}

// CatchCritErr() will handle critical errors
func CatchCritErr(err error) {
	if err != nil && !Silent {
		fmt.Printf("CRITICAL: %s\n", err)
	}
	os.Exit(1)
}
