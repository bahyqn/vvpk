package vvpk

import (
	"bufio"
	"bytes"
	"encoding/binary"
	"fmt"
	"io"
	"os"
	"regexp"
	"strings"
	"unsafe"
)

type VDFS struct {
	// Store key value
	Stack [][]string
	// index of the lastest { in stack
	LastKeyIdx int
	// Record index of stack's element while it met {
	SubKeyIdx []int
	// Store map key, unique path
	PendingKey []string
	KeyLength  int
}

func NewVDFS() *VDFS {
	return &VDFS{
		Stack:      [][]string{},
		LastKeyIdx: -1,
		SubKeyIdx:  []int{},
		PendingKey: []string{},
		KeyLength:  -1,
	}
}

func (vdfs *VDFS) newSubMap(root map[string]any) {
	tempMap := root

	if vdfs.KeyLength >= 0 {
	}
	for i := 0; i <= vdfs.KeyLength; i++ {

		t, ok := tempMap[vdfs.PendingKey[i]].(map[string]any)

		if !ok {
			panic("newSubMap")
		}
		tempMap = t
	}

	if vdfs.KeyLength == -1 {
		for _, el := range vdfs.Stack {
			// fmt.Println("-1 ", el)
			tempMap[el[0]] = el[1]
		}
	}

	if vdfs.KeyLength >= 0 {
		for _, el := range vdfs.Stack[vdfs.SubKeyIdx[vdfs.KeyLength]:] {
			// fmt.Println(el)
			tempMap[el[0]] = el[1]
		}

		vdfs.Stack = vdfs.Stack[:vdfs.LastKeyIdx]

		vdfs.PendingKey = vdfs.PendingKey[:vdfs.KeyLength]
		vdfs.KeyLength -= 1
	}
}

func (vdfs *VDFS) newMapKey(root map[string]any) {
	if vdfs.KeyLength == 0 && vdfs.LastKeyIdx >= 0 {
		root[vdfs.PendingKey[0]] = make(map[string]any)
		return
	}

	tempMap := root

	if vdfs.KeyLength > 0 {
		for i := 0; i < vdfs.KeyLength; i++ {

			t, ok := tempMap[vdfs.PendingKey[i]].(map[string]any)
			if !ok {
				panic("newMapKey ")
			}
			tempMap = t
		}
		tempMap[vdfs.PendingKey[vdfs.KeyLength]] = make(map[string]any)
		vdfs.SubKeyIdx = append(vdfs.SubKeyIdx, vdfs.LastKeyIdx)
	}
}

var tokenRegexp = regexp.MustCompile(`"([^"]*)"|(\{|\}|\S+)`)

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

		if len(f_slice) <= 1 {
			return fmap, fmt.Errorf("Files format error")
		}

		if f_slice[1] != "txt" {
			return fmap, fmt.Errorf("File extension is not txt")
		}
		fmap[el] = ""
	}
	return fmap, nil
}

func StringToMap(text string) (map[string]any, error) {
	if text == "" {
		return map[string]any{}, nil
	}

	start := strings.Index(text, "{")
	end := strings.LastIndex(text, "}")

	if start != -1 && end != -1 && start < end {
		str := strings.TrimSpace(text[start : end+1])
		// str = removeComments(str)
		return ParseVDFSinglePass(str)
	}

	return map[string]any{}, fmt.Errorf("xxxxxxxxxx")
}

func removeComments(input string) string {
	var builder strings.Builder
	scanner := bufio.NewScanner(strings.NewReader(input))

	for scanner.Scan() {
		line := scanner.Text()

		inQuote := false
		commentIdx := -1

		for i := 0; i < len(line); i++ {
			if line[i] == '"' {
				inQuote = !inQuote
			} else if !inQuote && i+1 < len(line) && line[i] == '/' && line[i+1] == '/' {
				commentIdx = i
				break
			}
		}

		if commentIdx != -1 {
			line = line[:commentIdx]
		}

		line = strings.TrimSpace(line)
		if line != "" {
			builder.WriteString(line)
			builder.WriteString("\n")
		}
	}
	return builder.String()
}

func kvToMap(cleanInput string) map[string]string {
	result := make(map[string]string)

	tokenRegex := regexp.MustCompile(`"([^"]*)"|(\S+)`)

	scanner := bufio.NewScanner(strings.NewReader(cleanInput))
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())

		if line == "{" || line == "}" || line == "" {
			continue
		}

		matches := tokenRegex.FindAllStringSubmatch(line, -1)
		tokens := make([]string, 0, len(matches))

		for _, m := range matches {
			if m[1] != "" {
				tokens = append(tokens, m[1])
			} else if m[2] != "" {
				tokens = append(tokens, m[2])
			}
		}

		if len(tokens) >= 2 {
			key := tokens[0]
			val := tokens[1]
			result[key] = val
		}
	}

	return result
}

// func ParseVDFSinglePass(text string) (map[string]any, error) {
func ParseVDFSinglePass(text string) (map[string]any, error) {
	// vdfs := VDFS{}
	vdfs := NewVDFS()
	root := make(map[string]any)
	scanner := bufio.NewScanner(strings.NewReader(text))

	for scanner.Scan() {
		line := scanner.Text()

		inQuote := false
		for i := 0; i < len(line); i++ {
			if line[i] == '"' {
				inQuote = !inQuote
			} else if !inQuote && i+1 < len(line) && line[i] == '/' && line[i+1] == '/' {
				line = line[:i]
				break
			}
		}

		switch strings.TrimSpace(line) {
		case "{":
			if vdfs.KeyLength >= 0 {
				vdfs.newMapKey(root)
			}
		case "}":
			// vdfs.PendingKey = vdfs.PendingKey[:vdfs.KeyLength-1]
			vdfs.newSubMap(root)
		default:
			// fmt.Println(line)
			st := strings.TrimSpace(line)
			tokenSlice := extractTokensWithRegexp(st)

			switch len(tokenSlice) {
			case 1:
				vdfs.KeyLength += 1
				vdfs.PendingKey = append(vdfs.PendingKey, tokenSlice[0])
				vdfs.SubKeyIdx = append(vdfs.SubKeyIdx, vdfs.LastKeyIdx)
			case 2:
				vdfs.Stack = append(vdfs.Stack, tokenSlice)
				vdfs.LastKeyIdx += 1
			}
		}
	}
	// fmt.Printf("root ---------> %+v", root)
	return root, nil
}

func extractTokensWithRegexp(line string) []string {
	matches := tokenRegexp.FindAllStringSubmatch(line, -1)
	tokens := make([]string, 0, len(matches))

	for _, m := range matches {
		var tok string

		if strings.HasPrefix(m[0], `"`) {
			tok = m[1]
		} else {
			tok = m[2]
		}

		tok = strings.TrimSpace(tok)

		if tok != "" || strings.HasPrefix(m[0], `"`) {
			tokens = append(tokens, tok)
		}
	}
	return tokens
}
