package main

/*
#cgo CFLAGS: -I/opt/local/include
#cgo LDFLAGS: -L/opt/local/lib -lz -llzma
#include <zlib.h>
#include <lzma.h>
int zlib_decomp(void *out, int olen, void *in, int ilen) {
	uLongf zolen = olen;
	int err = uncompress(out, &zolen, in, ilen);
	if (err)
		return 0;
	return zolen;
}
int lzma_comp(void *out, int olen, void *in, int ilen) {
	size_t zolen = 0;
	lzma_ret err = lzma_easy_buffer_encode(LZMA_PRESET_DEFAULT,
		LZMA_CHECK_CRC32, NULL, in, ilen, out, &zolen, olen);
	if (err)
		return 0;
	return zolen;
}
*/
import "C"

import (
	"encoding/binary"
	"fmt"
	"io"
	"os"
	"unsafe"
)

const MagicLen = 128
const Magic = "#!/bin/sh\n"

type Uzip struct {
	io io.ReadSeeker
	bsize, blocks uint32
	offsets []uint64
}

func NewUzip(name string) *Uzip {
	u := new(Uzip)
	u.io, _ = os.Open(name)
	u.io.Seek(MagicLen, 0)
	binary.Read(u.io, binary.BigEndian, &u.bsize)
	binary.Read(u.io, binary.BigEndian, &u.blocks)
	u.offsets = make([]uint64, u.blocks + 1)
	binary.Read(u.io, binary.BigEndian, u.offsets)
	return u
}

func (u *Uzip) Seek(block int) {
	u.io.Seek(int64(u.offsets[block]), 0)
}

// assume we're already at the start of the block
func (u *Uzip) Read(block int) []byte {
	start := u.offsets[block]
	len := u.offsets[block+1] - start
	buf := make([]byte, len)
	u.io.Read(buf)
	return buf
}

func (u *Uzip) Decomp(in []byte) []byte {
	out := make([]byte, u.bsize)
	err := C.zlib_decomp(unsafe.Pointer(&out[0]), C.int(u.bsize),
	 	unsafe.Pointer(&in[0]), C.int(len(in)))
	if uint32(err) != u.bsize {
		panic("Decompression error")
	}
	return out
}


const UlzmaVers = "#L3"

type Ulzma struct {
	io io.WriteSeeker
	bsize, blocks uint32
	offsets []uint64
	cur int
}

func NewUlzma(name string, uz *Uzip) *Ulzma {
	io, _ := os.Create(name)
	offsets := make([]uint64, len(uz.offsets))
	offsets[0] = 0
	io.Seek(int64(MagicLen + 8 + 8 * len(offsets)), 0)
	return &Ulzma{io, uz.bsize, uz.blocks, offsets, 0}
}

func (u *Ulzma) Comp(in []byte) []byte {
	out := make([]byte, len(in) * 2)
	err := C.lzma_comp(unsafe.Pointer(&out[0]), C.int(len(out)),
	 	unsafe.Pointer(&in[0]), C.int(len(in)))
	if err == 0 {
		panic("Compression error")
	}
	return out[:err]
}

func (u *Ulzma) Append(buf []byte) {
	u.offsets[u.cur + 1] = u.offsets[u.cur] + uint64(len(buf))
	u.cur++
	u.io.Write(buf)
}

func (u *Ulzma) Finish() {
	u.io.Seek(0, 0)
	io.WriteString(u.io, Magic)
	io.WriteString(u.io, UlzmaVers)
	u.io.Seek(MagicLen, 0)
	binary.Write(u.io, binary.BigEndian, u.bsize)
	binary.Write(u.io, binary.BigEndian, u.blocks)
	binary.Write(u.io, binary.BigEndian, u.offsets)
}


func main() {
	uz := NewUzip(os.Args[1])
	ul := NewUlzma(os.Args[2], uz)
	
	for b := 0; b < uz.blocks; b++ {
		fmt.Println(b)
		zb := uz.Read(b)
		buf := uz.Decomp(zb)
		lb := ul.Comp(buf)
		ul.Append(lb)
	}
	ul.Finish()
}

func grrrr() {
	fmt.Print(1)
}