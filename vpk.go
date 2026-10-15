package vvpk

import (
	"bufio"
	"cmp"
	"encoding/binary"
	"fmt"
	"io"
	"os"
	"slices"
	"strings"
)

type VpkSummary struct {
	Addoninfo  string
	Missions   map[string]string
	VpkVersion uint32
	TreeSize   uint32
}

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
	// PreloadBytes      [2]byte
	PrelaodBytes uint16
	Preload      []byte
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
func OpenVpk(path string) VpkSummary {
	// file_content := map[string]string{}
	// files := []string{"addoninfo", "missions/*.txt"}

	// if len(files) == 0 {
	// 	files = append(files, "addoninfo.txt")
	// 	files = append(files, "missions/*.txt")
	// }

	// fmap, err := sliceToMap(files)

	vpkSummary := VpkSummary{
		Missions: make(map[string]string),
	}

	// fmt.Println(fmap)

	// if err != nil {
	// 	return file_content
	// }

	file, err := os.Open(path)
	if err != nil {
		return vpkSummary
	}
	defer file.Close()

	// version, err := detectVersion(file)
	// version, err := detectVersion(file)
	// if err != nil {
	// 	return vpkSummary
	// }

	// fmap["version"] = strconv.FormatUint(uint64(version), 10)
	// vpkSummary.VpkVersion = version
	_ = parseVPK(file, &vpkSummary)
	return vpkSummary
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

func parseVPK(f *os.File, vpkSummary *VpkSummary) error {
	var headerBuf [12]byte

	if _, err := io.ReadFull(f, headerBuf[:]); err != nil {
		return fmt.Errorf("read v1 header failed: %w", err)
	}

	vpkVersion := binary.LittleEndian.Uint32(headerBuf[:][4:8])
	treeSize := binary.LittleEndian.Uint32(headerBuf[:][8:12])

	vpkSummary.VpkVersion = vpkVersion
	vpkSummary.TreeSize = treeSize

	if treeSize <= 0 {
		return fmt.Errorf("Read treeSize went wrong")
	}

	treeReader := io.LimitReader(f, int64(treeSize))
	file_archive := []Metadata{}

	err := walkV1Entries(treeReader, func(m Metadata) error {

		k := strings.ToLower(m.Filename)
		// currentPos, _ := f.Seek(0, io.SeekCurrent)
		// fmt.Printf("seek: %d ---> %s\n", currentPos, filepath.Join(m.Path, m.Filename+"."+m.Extension))

		if k == "addoninfo" {
			if m.PrelaodBytes > 0 {
				vpkSummary.Addoninfo = string(m.Preload)
			} else {
				file_archive = append(file_archive, m)
			}
			// fmt.Println("aaddoninfo seek: ", currentPos)
			// fmt.Printf("%+v\n", m)
		}

		if m.Path == "missions" {
			// fmt.Println("missions seek: ", currentPos)
			file_archive = append(file_archive, m)
		}
		return nil
	})

	// fmt.Println(len(file_archive))
	// fmt.Printf("---+> %+v\n", file_archive)
	// make sure the seek was in right index
	ensureDataSectionStart(f, 12+int64(treeSize))
	SortByOffset(file_archive)

	// fmt.Println(len(file_archive))
	// fmt.Printf("----11> %+v\n", file_archive)
	for _, el := range file_archive {
		// fmt.Printf("----> %+v\n\n", el)
		ensureDataSectionStart(f, 12+int64(treeSize)+int64(el.EntryOffset))

		buf := make([]byte, el.EntryLength)

		_, err := io.ReadFull(f, buf)

		if err != nil {
			continue
		}

		if el.Path == "" && strings.ToLower(el.Filename) == "addoninfo" && strings.HasPrefix(strings.ToLower(el.Extension), "tx") {
			vpkSummary.Addoninfo = string(buf)
			continue
		}
		// missions
		vpkSummary.Missions[strings.ToLower(el.Filename)] = string(buf)
	}

	if err != nil {
		return fmt.Errorf("Forloop files inside vpk went wrong")
	}

	return nil
}

// return addoninfo slice && idxMap(quickly find some vpk in slice)
func OpenAddonlist(path string) ([]string, map[string]int, error) {
	content := []string{}
	idxMap := map[string]int{}

	f, err := os.Open(path)

	if err != nil {
		return content, idxMap, fmt.Errorf("addonlist path is invalid")
	}
	defer f.Close()

	count := -1
	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		count += 1
		text := scanner.Text()
		content = append(content, text)

		key := ExtractVpkId(text)
		if key != "" {
			idxMap[key] = count
		}
	}

	if err := scanner.Err(); err != nil {
		return content, idxMap, fmt.Errorf("Error during scan %s", path)
	}
	return content, idxMap, nil
}

func UpdateModStatus(content []string, vpkId string, status string) error {
	if status != "0" && status != "1" {
		return fmt.Errorf("Attention: status only 0 or 1")
	}

	for i, line := range content {
		if !strings.Contains(line, vpkId) {
			continue
		}

		quoteCount := 0
		for pos := 0; pos < len(line); pos++ {
			if line[pos] != '"' {
				continue
			}

			quoteCount++
			if quoteCount == 3 {
				if pos+1 >= len(line) {
					return fmt.Errorf("invalid addonlist entry for %q", vpkId)
				}

				updated := []byte(line)
				updated[pos+1] = status[0]
				content[i] = string(updated)
				return nil
			}
		}

		return fmt.Errorf("invalid addonlist entry for %q", vpkId)
	}

	return nil
}

func ExtractVpkId(rawStr string) string {
	first := strings.IndexByte(rawStr, '"')
	if first == -1 {
		return ""
	}

	last := strings.IndexByte(rawStr[first+1:], '"')
	if last == -1 {
		return ""
	}

	quotedPath := rawStr[first+1 : first+1+last]
	filename := quotedPath[strings.LastIndexAny(quotedPath, `/\\`)+1:]
	if extension := strings.LastIndexByte(filename, '.'); extension >= 0 {
		filename = filename[:extension]
	}
	return filename
}

func SaveAddonlist(path string, rawSlice []string) error {
	content := strings.Join(rawSlice, "\n")
	if len(rawSlice) > 0 {
		content += "\n"
	}

	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		return fmt.Errorf("write addonlist %q failed: %w", path, err)
	}
	return nil
}
