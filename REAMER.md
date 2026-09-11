# vvpk

```
bsp  map
wav  sound
mdl  model

vmt  metarial
vtf  metarial

nut  script
```

## struct
```
type VpkArchive struct {
    // v1
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
    // preloadData
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

```

## Func
```
func OpenVpk(path string) (VpkArchive, error)

// It calculates the last file's(maximum offset) crc inside the vpk.
func VerifyBoundary(path string) (bool, error)

// It calculates all files's crc inside the vpk
func VerifyChecksums(path string) ([]FailedEntry, error)

// offset small to large
func SortByOffset(metadata []Metadata)
```