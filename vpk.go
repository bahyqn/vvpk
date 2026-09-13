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

type Filemap struct {
	Addoninfo string
	Missions  map[string]string
	Version   uint32
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
func OpenVpk(path string) Filemap {
	// file_content := map[string]string{}
	// files := []string{"addoninfo", "missions/*.txt"}

	// if len(files) == 0 {
	// 	files = append(files, "addoninfo.txt")
	// 	files = append(files, "missions/*.txt")
	// }

	// fmap, err := sliceToMap(files)

	fmap := Filemap{
		Missions: make(map[string]string),
	}

	// fmt.Println(fmap)

	// if err != nil {
	// 	return file_content
	// }

	file, err := os.Open(path)
	if err != nil {
		return fmap
	}
	defer file.Close()

	// version, err := detectVersion(file)
	version, err := detectVersion(file)
	if err != nil {
		return fmap
	}

	// fmap["version"] = strconv.FormatUint(uint64(version), 10)
	fmap.Version = version
	_ = parseVPK(file, &fmap)
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

func parseVPK(f *os.File, fmap *Filemap) error {
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

		k := strings.ToLower(m.Filename)
		// currentPos, _ := f.Seek(0, io.SeekCurrent)
		// fmt.Printf("seek: %d ---> %s\n", currentPos, filepath.Join(m.Path, m.Filename+"."+m.Extension))

		if k == "addoninfo" {
			if m.PrelaodBytes > 0 {
				fmap.Addoninfo = string(m.Preload)
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
			fmap.Addoninfo = string(buf)
			continue
		}
		// missions
		fmap.Missions[strings.ToLower(el.Filename)] = string(buf)
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
