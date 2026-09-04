package vvpk

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"io"
	"os"
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

func OpenVpk(path string) (VpkArchive, error) {
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
		return parseVPKv1(file)
	case 2:
		return parseVPKv2(file)
	default:
		return VpkArchive{}, fmt.Errorf("unsupported vpk version: %d", version)
	}
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
