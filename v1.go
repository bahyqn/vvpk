package vvpk

import (
	"encoding/binary"
	"fmt"
	"hash/crc32"
	"io"
	"os"
)

// func parseVPKv1(r io.Reader) (VpkArchive, error) {
func parseVPKv1Dev(f *os.File) (VpkArchive, error) {
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
	err := walkV1Entries(treeReader, func(entry Metadata) error {
		vpkArchive.Entries = append(vpkArchive.Entries, entry)
		return nil
	})
	if err != nil {
		return vpkArchive, err
	}

	// var offset, length uint32

	// for _, el := range vpkArchive.Entries {
	// 	if el.Filename == "addoninfo" {
	// 		offset = el.EntryOffset
	// 		length = el.EntryLength
	// 	}
	// }

	// _, err = f.Seek(int64(offset), io.SeekCurrent)

	// if err != nil {
	// 	panic("1111111")
	// }

	// buf := make([]byte, length)
	// if _, err = io.ReadFull(f, buf); err != nil {
	// 	panic("222222")
	// }

	// fmt.Println(string(buf))
	return vpkArchive, nil
}

func VerifyBoundary_v1Dev(f *os.File) (bool, error) {
	var headerBuf [12]byte

	if _, err := io.ReadFull(f, headerBuf[:]); err != nil {
		return false, fmt.Errorf("read v1 header failed: %w", err)
	}

	treeSize := binary.LittleEndian.Uint32(headerBuf[8:12])
	treeReader := io.LimitReader(f, int64(treeSize))

	info, err := f.Stat()
	if err != nil {
		return false, fmt.Errorf("stat vpk failed: %w", err)
	}

	var metadata Metadata
	err = walkV1Entries(treeReader, func(entry Metadata) error {
		if entry.ArchiveIndex == 0x7fff &&
			int64(entry.EntryOffset)+int64(entry.EntryLength) > int64(metadata.EntryOffset)+int64(metadata.EntryLength) {
			metadata = entry
		}
		return nil
	})
	if err != nil {
		return false, err
	}

	return calculateMaxOffsetCRC(f, metadata, info.Size(), int64(12+treeSize))
}

func walkV1Entries(r io.Reader, visit func(Metadata) error) error {
	var ext, fpath string
	state := 0

	for {
		name, err := readNullTerminatedString(r)
		if err != nil {
			return err
		}

		switch state {
		case 0:
			if name == "" {
				return nil
			}
			ext = name
			state = 1
		case 1:
			if name == "" {
				state = 0
				continue
			}
			fpath = name
			if fpath == " " {
				fpath = ""
			}
			state = 2
		case 2:
			if name == "" {
				state = 1
				continue
			}
			if name == " " {
				continue
			}

			entry, err := readV1Entry(r, ext, fpath, name)
			if err != nil {
				return err
			}
			if err := visit(entry); err != nil {
				return err
			}
		}
	}
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

	// fmt.Println("ext ", ext)
	// fmt.Println("path ", path)
	// fmt.Println("filename", filename)
	// fmt.Println("archiveindex ", archiveIndex)
	// fmt.Println("preload ", preload)
	// fmt.Println("offset ", entryOffset)
	// fmt.Println("length ", entryLength)

	return Metadata{
		Extension:    ext,
		Path:         path,
		Filename:     filename,
		Checksum:     checksum,
		PrelaodBytes: preloadLength,
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

	var calculated_crc uint32
	calculated_crc = crc32.ChecksumIEEE(metadata.Preload)

	// hasher := crc32.NewIEEE()

	// if _, err := hasher.Write(metadata.Preload); err != nil {
	// 	return false, fmt.Errorf("hash preload data failed: %w", err)
	// }

	if err := ensureDataSectionStart(f, headerSiize); err != nil {
		return false, err
	}
	if metadata.EntryLength > 0 {

		if _, err := f.Seek(headerSiize+int64(metadata.EntryOffset), io.SeekStart); err != nil {
			return false, fmt.Errorf("seek to target payload failed: %w", err)
		}

		// if _, err := io.CopyN(hasher, f, int64(metadata.EntryLength)); err != nil {
		// 	return false, fmt.Errorf("read payload data failed: %w", err)
		// }

		buf := make([]byte, metadata.EntryLength)
		if _, err := io.ReadFull(f, buf); err != nil {
			return false, fmt.Errorf("Failed to read bytes wiith maxOffset")
		}

		calculated_crc = crc32.ChecksumIEEE(buf)
	}

	if calculated_crc != metadata.Checksum {
		return false, fmt.Errorf("crc mismatch: calculated 0x%08X != expected 0x%08X", calculated_crc, metadata.Checksum)
	}
	return true, nil
}

func calculateEntryCRC_v1Dev(f *os.File, failedEntries []FailedEntry) ([]FailedEntry, error) {
	var headerBuf [12]byte

	if _, err := io.ReadFull(f, headerBuf[:]); err != nil {
		return failedEntries, fmt.Errorf("read v1 header failed: %w", err)
	}

	treeSize := binary.LittleEndian.Uint32(headerBuf[8:12])
	treeReader := io.LimitReader(f, int64(treeSize))

	var vpkArchive VpkArchive
	err := walkV1Entries(treeReader, func(entry Metadata) error {
		vpkArchive.Entries = append(vpkArchive.Entries, entry)
		return nil
	})

	if err != nil {
		return failedEntries, err
	}

	SortByOffset(vpkArchive.Entries)

	if err = ensureDataSectionStart(f, int64(12+treeSize)); err != nil {
		return failedEntries, err
	}

	for _, item := range vpkArchive.Entries {
		// fmt.Printf("%d: \t %s \t %s \t %s \t ---> offset: %d, length: %d\n", idx, item.Extension, item.Path, item.Filename, item.EntryOffset, item.EntryLength)

		same, crc, _ := calculateCRC(f, item)

		// fmt.Printf("crc: %d\ncalculated_crc: %d\n\n", item.Checksum, crc)
		if !same {
			failedEntries = append(failedEntries, FailedEntry{
				MetaData:      item,
				CalculatedCRC: crc,
				Error:         err,
			})
			// fmt.Printf("expected crc: %d, calculated crc: %d\n\n", item.Checksum, crc)
		}
	}

	return failedEntries, nil
}
