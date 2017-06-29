package par1

import (
	"bytes"
	"encoding/binary"
	"errors"
	"reflect"
	"unicode/utf16"
	"unicode/utf8"
)

type fileEntryHeader struct {
	EntryBytes   uint64
	Status       uint64
	FileBytes    uint64
	Hash         [16]byte
	SixteenKHash [16]byte
}

func sizeOfFileEntryHeader() uint64 {
	return uint64(reflect.TypeOf(fileEntryHeader{}).Size())
}

type fileEntry struct {
	header   fileEntryHeader
	filename string
}

func decodeUTF16LEString(bs []byte) string {
	u16s := make([]uint16, len(bs)/2)
	for i := 0; i < len(u16s); i++ {
		u16s[i] = uint16(bs[2*i]) + uint16(bs[2*i+1])<<8
	}

	runes := utf16.Decode(u16s)

	var runeBuf [4]byte
	var buf bytes.Buffer
	for i := 0; i < len(runes); i++ {
		n := utf8.EncodeRune(runeBuf[:], runes[i])
		buf.Write(runeBuf[:n])
	}

	return buf.String()
}

func encodeUTF16LEString(s string) []byte {
	var runes []rune
	for _, r := range s {
		runes = append(runes, r)
	}

	u16s := utf16.Encode(runes)

	bs := make([]byte, 2*len(u16s))
	for i := 0; i < len(u16s); i++ {
		bs[2*i] = byte(u16s[i])
		bs[2*i+1] = byte(u16s[i] >> 8)
	}
	return bs
}

func readFileEntry(buf *bytes.Buffer) (fileEntry, error) {
	var header fileEntryHeader
	err := binary.Read(buf, binary.LittleEndian, &header)
	if err != nil {
		return fileEntry{}, err
	}

	filenameByteCount := header.EntryBytes - sizeOfFileEntryHeader()
	if filenameByteCount <= 0 || filenameByteCount%2 != 0 {
		return fileEntry{}, errors.New("invalid entry byte count")
	}
	if filenameByteCount > uint64(buf.Len()) {
		return fileEntry{}, errors.New("byte count mismatch")
	}

	filename := decodeUTF16LEString(buf.Next(int(filenameByteCount)))

	return fileEntry{header, filename}, nil
}

func writeFileEntry(entry fileEntry) ([]byte, error) {
	encodedFilename := encodeUTF16LEString(entry.filename)
	header := entry.header
	header.EntryBytes = sizeOfFileEntryHeader() + uint64(len(encodedFilename))

	buf := bytes.NewBuffer(nil)
	err := binary.Write(buf, binary.LittleEndian, header)
	if err != nil {
		return nil, err
	}
	return append(buf.Bytes(), encodedFilename...), nil
}
