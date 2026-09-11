package vvpk

import (
	"cmp"
	"fmt"
	"hash/crc32"
	"io"
	"os"
	"slices"
)

type VpkArchive struct {
	Signature uint32
	Version   uint32
	TreeSize  uint32
	Entries   []Metadata

	// v2
	FileDataSectionSize   uint32
	ArchiveMD5SectionSize uint32
	OtherMD5SectionSize   uint32
	SignatureSectionSize  uint32
}

type Metadata struct {
	Extension string
	Path      string
	Filename  string
	// Checksum     [4]byte
	Checksum uint32
	// Preload      [2]byte
	Preload []byte
	// ArchiveIndex [2]byte
	ArchiveIndex uint16
	// EntryOffset  [4]byte
	EntryOffset uint32
	// EntryLength [4]byte
	EntryLength uint32
	// Tail        [2]byte
	Tail uint16
}

type FailedEntry struct {
	MetaData      Metadata
	CalculatedCRC uint32
	Error         error
}

func (vpk VpkArchive) LengthValidate() uint32 {
	var len uint32

	len += 12
	len += vpk.TreeSize

	for _, el := range vpk.Entries {
		len += el.EntryLength
	}

	return len
}

// Return strings such as addoninfo.txt, missions/*.txt
func OpenVpk(path string) map[string]string {
	file_content := map[string]string{}
	files := []string{"addoninfo.txt", "missions/*.txt"}

	// if len(files) == 0 {
	// 	files = append(files, "addoninfo.txt")
	// 	files = append(files, "missions/*.txt")
	// }

	fmap, err := sliceToMap(files)

	if err != nil {
		return file_content
	}

	file, err := os.Open(path)
	if err != nil {
		return file_content
	}
	defer file.Close()

	// version, err := detectVersion(file)
	_, err = detectVersion(file)
	if err != nil {
		return file_content
	}

	_ = parseVPKv1(file, fmap)
	return fmap
}

func OpenVpkDev(path string) (VpkArchive, error) {
	file, err := os.Open(path)
	if err != nil {
		return VpkArchive{}, fmt.Errorf("open vpk failed: %w", err)
	}
	defer file.Close()

	version, err := detectVersion(file)
	if err != nil {
		return VpkArchive{}, err
	}

	// if _, err := file.Seek(0, io.SeekStart); err != nil {
	// 	return VpkArchive{}, fmt.Errorf("seek vpk header failed: %w", err)
	// }

	switch version {
	case 1:
		return parseVPKv1Dev(file)
	case 2:
		return parseVPKv2Dev(file)
	default:
		return VpkArchive{}, fmt.Errorf("unsupported vpk version: %d", version)
	}
}

func SortByOffset(metadata []Metadata) {
	slices.SortFunc(metadata, func(a, b Metadata) int {
		return cmp.Compare(a.EntryOffset, b.EntryOffset)
	})
}

func ensureDataSectionStart(f *os.File, expectedOffset int64) error {
	currentPos, _ := f.Seek(0, io.SeekCurrent)

	if currentPos == expectedOffset {
		return nil
	}

	if _, err := f.Seek(expectedOffset, io.SeekStart); err != nil {
		return fmt.Errorf("failed to align stream to offset %d: %w", expectedOffset, err)
	}

	return nil
}

func calculateCRC(f *os.File, metadata Metadata) (bool, uint32, error) {
	var calculated_crc uint32

	if metadata.ArchiveIndex != 0x7fff {
		calculated_crc = crc32.ChecksumIEEE(metadata.Preload)
	} else {
		tmp_buf := make([]byte, metadata.EntryLength)

		if _, err := io.ReadFull(f, tmp_buf); err != nil {
			return false, 0, fmt.Errorf("seek went wrong place.")
		}

		calculated_crc = crc32.ChecksumIEEE(tmp_buf)
	}

	if metadata.Checksum != calculated_crc {
		return false, calculated_crc, fmt.Errorf("crc was not same")
	}
	return true, calculated_crc, nil
}
