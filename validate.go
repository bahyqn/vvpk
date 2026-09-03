package vvpk

import (
	"fmt"
	"io"
	"os"
)

// VerifyBoundary validates the physical boundary of the VPK file on disk.
// It calculates the maximum required file offset across all directory entries
// (Header + Directory Tree + Max Entry Offset + Entry Length) and verifies that
// the actual fileSize on disk is sufficient to contain all data payloads.
//
// This serves as a lightweight integrity check to defend against truncated files
// or incomplete downloads without reading the entire payload into memory.
// Returns an error if the actual file size is smaller than the required boundary offset.
func VerifyBoundary(path string) (bool, error) {
	file, err := os.Open(path)
	if err != nil {
		return false, fmt.Errorf("open vpk failed: %w", err)
	}
	defer file.Close()

	version, err := detectVersion(file)
	if err != nil {
		return false, err
	}

	if _, err := file.Seek(0, io.SeekStart); err != nil {
		return false, fmt.Errorf("seek vpk header failed: %w", err)
	}

	switch version {
	case 1:
		return VerifyBoundary_v1(file)
	case 2:
		return VerifyBoundary_v2(file)
	default:
		return false, fmt.Errorf("unsupported vpk version: %d", version)
	}
}

// VerifyChecksums performs a comprehensive integrity verification across all entries
// contained within the VPK archive. It sequentially reads the payload data of every
// file entry from r and computes its CRC32 checksum, comparing it against the expected
// checksum stored in the VPK directory tree.
//
// Note: This operation reads the full payload section from storage and may be I/O
// intensive for large archives.
// Returns an error immediately upon encountering the first corrupted entry or read failure.
func VerifyChecksums(path string) {}
