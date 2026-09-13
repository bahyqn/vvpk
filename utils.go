package vvpk

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"hash/crc32"
	"io"
	"os"
	"strings"
	"unsafe"
)


func detectVersion(f *os.File) (uint32, error) {
	var headerBuf [12]byte

	if _, err := io.ReadFull(f, headerBuf[:]); err != nil {
		return 0, fmt.Errorf("read header failed: %w", err)
	}

	signature := binary.LittleEndian.Uint32(headerBuf[:4])
	version := binary.LittleEndian.Uint32(headerBuf[4:8])

	if signature != 0x55AA1234 {
		return 0, fmt.Errorf("invalid vpk signature: 0x%08X", signature)
	}

	if version != 1 && version != 2 {
		return 0, fmt.Errorf("unsupported vpk version: %d", version)
	}

	if _, err := f.Seek(0, io.SeekStart); err != nil {
		return 0, fmt.Errorf("seek vpk header failed: %w", err)
	}

	return version, nil
}

func readPreloadBytes(preload *[]byte) ([]byte, error) {
	idx := bytes.IndexByte(*preload, 0)
	if idx == -1 {
		return nil, fmt.Errorf("null terminator not found in buffer")
	}

	result := (*preload)[:idx]
	return result, nil
}

func readNullTerminatedString(r io.Reader) (string, error) {
	var buf []byte

	for {
		b := make([]byte, 1)

		_, err := r.Read(b)

		if err != nil {
			return "", err
		}

		if b[0] == 0 {
			break
		}
		buf = append(buf, b[0])
	}
	return string(buf), nil
}

func readUnit[T ~uint8 | ~uint16 | ~uint32 | ~uint64](r io.Reader) (T, error) {
	var zero T
	size := int(unsafe.Sizeof(zero))

	buf := make([]byte, size)
	if _, err := io.ReadFull(r, buf); err != nil {
		return zero, err
	}

	switch any(zero).(type) {
	case uint8:
		return T(buf[0]), nil
	case uint16:
		return T(binary.LittleEndian.Uint16(buf)), nil
	case uint32:
		return T(binary.LittleEndian.Uint32(buf)), nil
	case uint64:
		return T(binary.LittleEndian.Uint64(buf)), nil
	default:
		return zero, fmt.Errorf("unsupported type")
	}
}

func sliceToMap(files []string) (map[string]string, error) {
	fmap := map[string]string{}

	for _, el := range files {
		f_slice := strings.Split(el, ".")
		
		// if len(f_slice) <= 1 {
		// 	return fmap, fmt.Errorf("Files format error")
		// }
		sliceLength := len(f_slice)

		if sliceLength > 1&& f_slice[1] != "txt" {
			return fmap, fmt.Errorf("File extension is not txt")
		}
		// fmt.Println(el)
		fmap[el] = ""
		// fmt.Println(fmap)
	}
	return fmap, nil
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
