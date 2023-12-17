// Package executable provides a function to check if a file is an executable file.
package executable

import (
	"bytes"
	"debug/elf"
	"debug/macho"
	"encoding/binary"
	"fmt"
	"io"
)

func readChunk(reader io.Reader, chunkSize int64) ([]byte, error) {
	buf := new(bytes.Buffer)
	_, err := io.CopyN(buf, reader, chunkSize)
	if err != nil && err != io.EOF {
		return []byte{}, err
	}
	return buf.Bytes(), nil
}

// IsExecutable check if source is an executable file
func IsExecutable(reader io.Reader) (bool, error) {
	buf, err := readChunk(reader, 4)
	if err != nil {
		return false, fmt.Errorf("failed to read file: %v", err)
	}
	if len(buf) < 4 {
		return false, nil
	}
	le := binary.LittleEndian.Uint32(buf)
	be := binary.BigEndian.Uint32(buf)

	return string(buf) == elf.ELFMAG || // elf - linux format exec file
			string(buf[:2]) == "MZ" || // .exe windows
			string(buf[:2]) == "#!" || // shebang
			macho.Magic32 == le || macho.Magic32 == be || macho.Magic64 == le || macho.Magic64 == be || macho.MagicFat == le || macho.MagicFat == be, // mach-o - mac format exec file
		nil
}
