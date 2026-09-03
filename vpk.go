package vvpk

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

func (vpk VpkArchive) LengthValidate() uint32 {
	var len uint32

	len += 12
	len += vpk.TreeSize

	for _, el := range vpk.Entries {
		len += el.EntryLength
	}

	return len
}
