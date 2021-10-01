package byteutil

import (
	"bytes"
	"encoding/binary"
	"strings"
	"sync"
	"unsafe"
)

func EncodeInt64ToBytes(id int64) []byte {
	b := make([]byte, 64)
	binary.BigEndian.PutUint64(b, uint64(id))
	return b
}

func BytesToString(b []byte) string {
	return *(*string)(unsafe.Pointer(&b))
}

var buffer = sync.Pool{
	New: func() interface{} {
		return &bytes.Buffer{}
	},
}

func GetBuffer() (b *bytes.Buffer) {
	ifc := buffer.Get()
	if ifc != nil {
		b = ifc.(*bytes.Buffer)
	}
	return
}

func PutBuffer(b *bytes.Buffer) {
	buffer.Put(b)
}

var builder = sync.Pool{
	New: func() interface{} {
		return &strings.Builder{}
	},
}

func GetBuilder() (b *strings.Builder) {
	ifc := builder.Get()
	if ifc != nil {
		b = ifc.(*strings.Builder)
	}
	return
}

func PutBuilder(b *strings.Builder) {
	builder.Put(b)
}
