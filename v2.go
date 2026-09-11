package vvpk

import (
	"fmt"
	"os"
)

func parseVPKv2Dev(f *os.File) (VpkArchive, error) {
	var vpkarch VpkArchive

	return vpkarch, nil
}

func VerifyBoundary_v2Dev(f *os.File) (bool, error) {

	return false, fmt.Errorf("")
}

func calculateEntryCRC_v2Dev(f *os.File, failedEntries []FailedEntry) ([]FailedEntry, error) {
	return failedEntries, nil
}
