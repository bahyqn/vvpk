package vvpk

import (
	"encoding/binary"
	"fmt"
	"hash/crc32"
	"io"
	"os"
)

// func parseVPKv1(r io.Reader) (VpkArchive, error) {
func parseVPKv1(f *os.File) (VpkArchive, error) {
	var headerBuf [12]byte

	if _, err := io.ReadFull(f, headerBuf[:]); err != nil {
		return VpkArchive{}, fmt.Errorf("read v1 header failed: %w", err)
	}

	sig := binary.LittleEndian.Uint32(headerBuf[:4])
	version := binary.LittleEndian.Uint32(headerBuf[4:8])
	treeSize := binary.LittleEndian.Uint32(headerBuf[8:12])

	vpkArchive := VpkArchive{
		Signature: sig,
		Version:   version,
		TreeSize:  treeSize,
	}

	treeReader := io.LimitReader(f, int64(treeSize))
	var ext, fpath, filename string

	for {
		if ext == "" {
			var err error
			ext, err = readNullTerminatedString(treeReader)
			if err != nil || ext == "" {
				break // 目录树解析完毕
			}
		}

		if fpath == "" {
			var err error
			fpath, err = readNullTerminatedString(treeReader)
			if err != nil {
				return vpkArchive, err
			}

			if fpath == "" {
				ext = ""
				continue
			}

			if fpath == " " {
				fpath = ""
			}
		}

		var err error
		filename, err = readNullTerminatedString(treeReader)
		if err != nil {
			return vpkArchive, err
		}

		if filename == "" {
			fpath = ""
			continue
		}

		if filename == " " {
			continue
		}

		crc, err := readUnit[uint32](treeReader)
		if err != nil {
			return vpkArchive, err
		}

		// preload, err := readUnit[uint16](treeReader)
		// if err != nil {
		// 	return vpkArchive, err
		// }

		preload := make([]byte, 2)
		if _, err := io.ReadFull(f, preload); err != nil {
			return vpkArchive, fmt.Errorf("read v1 preload data failed: %w", err)
		}

		archiveIndex, err := readUnit[uint16](treeReader)
		if err != nil {
			return vpkArchive, err
		}
		entryOffset, err := readUnit[uint32](treeReader)
		if err != nil {
			return vpkArchive, err
		}
		entryLength, err := readUnit[uint32](treeReader)
		if err != nil {
			return vpkArchive, err
		}
		terminator, err := readUnit[uint16](treeReader)
		if err != nil {
			return vpkArchive, err
		}

		vpkArchive.Entries = append(vpkArchive.Entries, Metadata{
			Extension:    ext,
			Path:         fpath,
			Filename:     filename,
			Checksum:     crc,
			Preload:      preload,
			ArchiveIndex: archiveIndex,
			EntryOffset:  entryOffset,
			EntryLength:  entryLength,
			Tail:         terminator,
		})
	}

	return vpkArchive, nil
}

func VerifyBoundary_v1(f *os.File) (bool, error) {
	var headerBuf [12]byte

	if _, err := io.ReadFull(f, headerBuf[:]); err != nil {
		return false, fmt.Errorf("read v1 header failed: %w", err)
	}

	treeSize := binary.LittleEndian.Uint32(headerBuf[8:12])

	treeReader := io.LimitReader(f, int64(treeSize))

	var metadata Metadata = Metadata{}
	var ext, fpath, filename string

	info, err := f.Stat()

	if err != nil {
		return false, fmt.Errorf("stat vpk failed: %w", err)
	}

	fileSize := info.Size()

	for {
		if ext == "" {
			var err error
			ext, err = readNullTerminatedString(treeReader)
			if err != nil || ext == "" {
				break
			}
		}

		if fpath == "" {
			var err error
			fpath, err = readNullTerminatedString(treeReader)
			if err != nil {
				return false, err
			}

			if fpath == "" {
				ext = ""
				continue
			}

			if fpath == " " {
				fpath = ""
			}
		}

		var err error
		filename, err = readNullTerminatedString(treeReader)
		if err != nil {
			return false, err
		}

		if filename == "" {
			fpath = ""
			continue
		}

		if filename == " " {
			continue
		}

		entry, err := readV1Entry(treeReader, ext, fpath, filename)
		if err != nil {
			return false, err
		}

		if entry.ArchiveIndex == 0x7fff &&
			int64(entry.EntryOffset)+int64(entry.EntryLength) > int64(metadata.EntryOffset)+int64(metadata.EntryLength) {
			metadata = entry
		}
	}

	finish, err := calculateMaxOffsetCRC(f, metadata, int64(fileSize), int64(12+treeSize))

	if !finish {
		return false, err
	}

	return true, nil
}

func readV1Entry(r io.Reader, ext, path, filename string) (Metadata, error) {
	checksum, err := readUnit[uint32](r)
	if err != nil {
		return Metadata{}, fmt.Errorf("read v1 checksum failed: %w", err)
	}

	preloadLength, err := readUnit[uint16](r)
	if err != nil {
		return Metadata{}, fmt.Errorf("read v1 preload length failed: %w", err)
	}
	archiveIndex, err := readUnit[uint16](r)
	if err != nil {
		return Metadata{}, fmt.Errorf("read v1 archive index failed: %w", err)
	}
	entryOffset, err := readUnit[uint32](r)
	if err != nil {
		return Metadata{}, fmt.Errorf("read v1 entry offset failed: %w", err)
	}
	entryLength, err := readUnit[uint32](r)
	if err != nil {
		return Metadata{}, fmt.Errorf("read v1 entry length failed: %w", err)
	}
	terminator, err := readUnit[uint16](r)
	if err != nil {
		return Metadata{}, fmt.Errorf("read v1 terminator failed: %w", err)
	}

	preload := make([]byte, preloadLength)
	if _, err := io.ReadFull(r, preload); err != nil {
		return Metadata{}, fmt.Errorf("read v1 preload data failed: %w", err)
	}

	return Metadata{
		Extension:    ext,
		Path:         path,
		Filename:     filename,
		Checksum:     checksum,
		Preload:      preload,
		ArchiveIndex: archiveIndex,
		EntryOffset:  entryOffset,
		EntryLength:  entryLength,
		Tail:         terminator,
	}, nil
}

func calculateMaxOffsetCRC(f *os.File, metadata Metadata, fileSize int64, headerSiize int64) (bool, error) {
	if metadata.ArchiveIndex != 0x7fff {
		return true, nil
	}

	requiredSize := headerSiize + int64(metadata.EntryOffset) + int64(metadata.EntryLength)

	// fmt.Println("headerSIze", headerSiize)
	// fmt.Println("offset", metadata.EntryOffset)
	// fmt.Println("length", metadata.EntryLength)

	if fileSize < requiredSize {
		return false, fmt.Errorf("vpk truncated: physical size (%d B) < required size (%d B)", fileSize, requiredSize)
	}

	hasher := crc32.NewIEEE()

	if _, err := hasher.Write(metadata.Preload); err != nil {
		return false, fmt.Errorf("hash preload data failed: %w", err)
	}

	if metadata.EntryLength > 0 {
		if _, err := f.Seek(headerSiize+int64(metadata.EntryOffset), io.SeekStart); err != nil {
			return false, fmt.Errorf("seek to target payload failed: %w", err)
		}

		if _, err := io.CopyN(hasher, f, int64(metadata.EntryLength)); err != nil {
			return false, fmt.Errorf("read payload data failed: %w", err)
		}
	}

	if calculatedCRC := hasher.Sum32(); calculatedCRC != metadata.Checksum {
		return false, fmt.Errorf("crc mismatch: calculated 0x%08X != expected 0x%08X", calculatedCRC, metadata.Checksum)
	}
	return true, nil
}

func calculateEntryCRC(file *os.File) {

}
