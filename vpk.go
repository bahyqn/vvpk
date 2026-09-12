package vvpk

import (
	"bufio"
	"cmp"
	"encoding/binary"
	"fmt"
	"io"
	"os"
	"path"
	"slices"
	"strconv"
	"strings"
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

// Return map[string]string{ "addoninfo.txt": "", "missions: "", "version": "1 or 2"}
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
	version, err := detectVersion(file)
	if err != nil {
		return file_content
	}

	fmap["version"] = strconv.FormatUint(uint64(version), 10)
	_ = parseVPK(file, fmap)
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

func parseVPK(f *os.File, f_map map[string]string) error {
	var headerBuf [12]byte

	if _, err := io.ReadFull(f, headerBuf[:]); err != nil {
		return fmt.Errorf("read v1 header failed: %w", err)
	}

	treeSize := binary.LittleEndian.Uint32(headerBuf[:][8:12])

	if treeSize <= 0 {
		return fmt.Errorf("Read treeSize went wrong")
	}

	treeReader := io.LimitReader(f, int64(treeSize))
	file_archive := []Metadata{}

	err := walkV1Entries(treeReader, func(m Metadata) error {
		key := m.Filename + "." + m.Extension

		_, ok := f_map[key]

		if ok || strings.ToLower(m.Path) == "missions" {
			// 0x7FFF (32767)
			if m.ArchiveIndex != 32767 {
				return fmt.Errorf("ArchiveIndex is not 0x7FFF (32767)")
			}

			file_archive = append(file_archive, m)
		}
		return nil
	})

	// make sure the seek was in right index
	ensureDataSectionStart(f, 12+int64(treeSize))
	SortByOffset(file_archive)

	for _, el := range file_archive {
		buf := make([]byte, el.EntryLength)
		key := ""

		if el.Path != "" {
			key = el.Path + "/" + el.Filename + "." + el.Extension
		} else {
			key = el.Filename + "." + el.Extension
		}

		// fmt.Println(key)

		_, ok := f_map[key]
		machted, err := path.Match("missions/*.txt", key)
		if !ok && !machted {
			continue
		}

		// fmt.Println(el)
		// fmt.Println(key)
		// fmt.Println(machted, err)
		// fmt.Println()

		if err != nil {
			continue
		}

		if !machted {
			f_map["missions"] = ""
			delete(f_map, "missions/*.txt")
		}

		ensureDataSectionStart(f, 12+int64(treeSize)+int64(el.EntryOffset))

		if _, err := io.ReadFull(f, buf); err == nil {
			if machted {
				f_map["missions"] = string(buf)
				delete(f_map, "missions/*.txt")
			}

			if !machted && strings.ToLower(el.Path) != "missions" {
				f_map[key] = string(buf)
			}
		}
	}

	if err != nil {
		return fmt.Errorf("Forloop files inside vpk went wrong")
	}

	return nil
}

func OpenAddonlist(path string) ([]string, error) {
	content := []string{}
	f, err := os.Open(path)

	if err != nil {
		return content, fmt.Errorf("addonlist path is invalid")
	}
	defer f.Close()

	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		content = append(content, scanner.Text())
	}

	if err := scanner.Err(); err != nil {
		return content, fmt.Errorf("Error during scan %s", path)
	}
	return content, nil
}

func UpdateModStatus(content []string, vpkId string, state string) {
	for elIdx, el := range content {
		if strings.Contains(el, vpkId) {

			b := []byte(el)
			count := 0

			for idx, str := range el {
				// fmt.Printf("%d ---> %d, %s\n", idx, str, string(str))
				// "
				if str == 34 {
					count += 1

					if count == 3 {
						switch b[idx+1] {
						// "0"
						case 48:
							b[idx+1] = 49
							// "1"
						case 49:
							b[idx+1] = 48
						}
						content[elIdx] = string(b)
						break
					}
				}
			}
			break
		}
	}
}
