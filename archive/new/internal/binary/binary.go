package binary

import (
	"encoding/binary"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"opcodeoracle/internal/asm/x86"
)

const (
	DefaultCOMSegment = 0x1000
	DefaultCOMOffset  = 0x0100
	DefaultCOMSP      = 0xfffe
	DefaultMZSegment  = 0x1000
)

type Binary struct {
	Data        []byte
	Origin      x86.FarAddress
	DefaultSP   x86.FarAddress
	EntryPoints []x86.FarAddress
}

func New(data []byte, origin x86.FarAddress, defaultSP x86.FarAddress, entryPoints []x86.FarAddress) Binary {
	return Binary{
		Data:        append([]byte(nil), data...),
		Origin:      origin,
		DefaultSP:   defaultSP,
		EntryPoints: append([]x86.FarAddress(nil), entryPoints...),
	}
}

func LoadFile(path string) (Binary, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return Binary{}, err
	}

	ext := strings.ToLower(filepath.Ext(path))
	if len(data) >= 2 && data[0] == 'M' && data[1] == 'Z' {
		return newFromMZData(data)
	}
	if ext == ".exe" {
		return Binary{}, fmt.Errorf("file has .exe extension but is not an MZ executable")
	}
	return newFromCOMData(data), nil
}

func (b Binary) ImageOffset(addr x86.FarAddress) (int, error) {
	target := addr.Linear()
	base := b.Origin.Linear()
	if target < base {
		return 0, fmt.Errorf("address %s is before origin %s", addr.String(), b.Origin.String())
	}

	diff := target - base
	if diff >= uint32(len(b.Data)) {
		return 0, fmt.Errorf("address %s is outside binary image", addr.String())
	}
	return int(diff), nil
}

func (b Binary) DataAt(addr x86.FarAddress) ([]byte, error) {
	imageOffset, err := b.ImageOffset(addr)
	if err != nil {
		return nil, err
	}
	return b.Data[imageOffset:], nil
}

func (b Binary) ReadWord(addr x86.FarAddress) (uint16, error) {
	imageOffset, err := b.ImageOffset(addr)
	if err != nil {
		return 0, err
	}
	if imageOffset+1 >= len(b.Data) {
		return 0, fmt.Errorf("address %s is outside binary image", addr.String())
	}
	return binary.LittleEndian.Uint16(b.Data[imageOffset : imageOffset+2]), nil
}

func (b Binary) ReadFarPointer(addr x86.FarAddress) (x86.FarAddress, error) {
	offset, err := b.ReadWord(addr)
	if err != nil {
		return x86.FarAddress{}, err
	}
	segment, err := b.ReadWord(x86.NewFarAddress(addr.Segment, addr.Offset+2))
	if err != nil {
		return x86.FarAddress{}, err
	}
	return x86.NewFarAddress(segment, offset), nil
}

func NewFromCOMFile(path string) (Binary, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return Binary{}, err
	}

	return newFromCOMData(data), nil
}

func NewFromMZFile(path string) (Binary, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return Binary{}, err
	}
	return newFromMZData(data)
}

func newFromCOMData(data []byte) Binary {
	start := x86.NewFarAddress(DefaultCOMSegment, DefaultCOMOffset)
	defaultSP := x86.NewFarAddress(DefaultCOMSegment, DefaultCOMSP)
	return New(data, start, defaultSP, []x86.FarAddress{start})
}

func newFromMZData(data []byte) (Binary, error) {
	if len(data) < 28 {
		return Binary{}, fmt.Errorf("mz file too small: %d bytes", len(data))
	}
	if data[0] != 'M' || data[1] != 'Z' {
		return Binary{}, fmt.Errorf("invalid mz signature")
	}

	lastPageBytes := binary.LittleEndian.Uint16(data[2:4])
	pageCount := binary.LittleEndian.Uint16(data[4:6])
	relocationCount := binary.LittleEndian.Uint16(data[6:8])
	headerParagraphs := binary.LittleEndian.Uint16(data[8:10])
	initialSP := binary.LittleEndian.Uint16(data[16:18])
	initialSS := binary.LittleEndian.Uint16(data[14:16])
	initialIP := binary.LittleEndian.Uint16(data[20:22])
	initialCS := binary.LittleEndian.Uint16(data[22:24])
	relocationTableOffset := binary.LittleEndian.Uint16(data[24:26])
	headerSize := int(headerParagraphs) * 16
	if headerSize > len(data) {
		return Binary{}, fmt.Errorf("invalid mz header size: %d", headerSize)
	}
	declaredFileSize, err := mzDeclaredFileSize(pageCount, lastPageBytes)
	if err != nil {
		return Binary{}, err
	}
	if declaredFileSize < headerSize {
		return Binary{}, fmt.Errorf("invalid mz file size: %d smaller than header %d", declaredFileSize, headerSize)
	}
	if declaredFileSize > len(data) {
		return Binary{}, fmt.Errorf("declared mz file size %d exceeds actual size %d", declaredFileSize, len(data))
	}

	image := append([]byte(nil), data[headerSize:declaredFileSize]...)
	if err := applyMZRelocations(image, data, relocationCount, relocationTableOffset, DefaultMZSegment); err != nil {
		return Binary{}, err
	}
	origin := x86.NewFarAddress(DefaultMZSegment, 0)
	entry := x86.NewFarAddress(DefaultMZSegment+initialCS, initialIP)
	defaultSP := x86.NewFarAddress(DefaultMZSegment+initialSS, initialSP)

	return New(image, origin, defaultSP, []x86.FarAddress{entry}), nil
}

func mzDeclaredFileSize(pageCount uint16, lastPageBytes uint16) (int, error) {
	if pageCount == 0 {
		return 0, fmt.Errorf("invalid mz page count: 0")
	}

	lastPageSize := int(lastPageBytes)
	if lastPageBytes == 0 {
		lastPageSize = 512
	}
	if lastPageSize > 512 {
		return 0, fmt.Errorf("invalid mz last page size: %d", lastPageBytes)
	}

	return (int(pageCount)-1)*512 + lastPageSize, nil
}

func applyMZRelocations(image []byte, exe []byte, relocationCount uint16, tableOffset uint16, loadSegment uint16) error {
	table := int(tableOffset)
	needed := table + int(relocationCount)*4
	if needed > len(exe) {
		return fmt.Errorf("invalid mz relocation table")
	}

	for i := 0; i < int(relocationCount); i++ {
		entryOffset := table + i*4
		offset := binary.LittleEndian.Uint16(exe[entryOffset : entryOffset+2])
		segment := binary.LittleEndian.Uint16(exe[entryOffset+2 : entryOffset+4])
		imageIndex := int(segment)*16 + int(offset)
		if imageIndex+1 >= len(image) {
			return fmt.Errorf("mz relocation out of range: %s", x86.NewFarAddress(segment, offset).String())
		}

		value := binary.LittleEndian.Uint16(image[imageIndex : imageIndex+2])
		value += loadSegment
		binary.LittleEndian.PutUint16(image[imageIndex:imageIndex+2], value)
	}

	return nil
}
