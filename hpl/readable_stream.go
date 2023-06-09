package hpl

import (
	"fmt"
	"github.com/dianpeng/moons/pl"
	"io"
)

type ReadableStream struct {
	Stream   io.ReadCloser
	cacheBuf []byte // maybe populated to locally cache the content of stream
	hasCache bool   // indicate whether the cacheBuf is valid or not
	closed   bool   // whether closed or not
}

func ValIsReadableStream(v pl.Val) bool {
	return v.Id() == ReadableStreamTypeId
}

func NewReadableStreamFromString(data string) *ReadableStream {
	return &ReadableStream{
		Stream:   neweofByteReadCloserFromString(data),
		cacheBuf: []byte(data),
		hasCache: true,
		closed:   false,
	}
}

func NewReadableStreamFromBuffer(b []byte) *ReadableStream {
	return &ReadableStream{
		Stream:   neweofByteReadCloser(b),
		cacheBuf: b,
		hasCache: true,
		closed:   false,
	}
}

func NewReadableStreamFromStream(stream io.ReadCloser) *ReadableStream {
	return &ReadableStream{
		Stream:   stream,
		cacheBuf: nil,
		hasCache: false,
		closed:   false,
	}
}

func (h *ReadableStream) ByteLength() int {
	if h.hasCache {
		return len(h.cacheBuf)
	} else {
		return -1
	}
}

func (h *ReadableStream) IsClose() bool {
	return h.closed
}

func (h *ReadableStream) Close() error {
	if h.closed {
		return nil
	}
	h.cacheBuf = []byte{}
	h.hasCache = true
	h.closed = true
	err := h.Stream.Close()
	h.Stream = &eofReadCloser{}
	return err
}

// Web standard like APS, which consumed the stream and make it closed after the
// consumption
func (h *ReadableStream) ConsumeAsString() (string, error) {
	b, err := h.readAll()
	if err != nil {
		return "", err
	}
	return string(b), nil
}

func (h *ReadableStream) readAll() ([]byte, error) {
	defer h.Close()
	if h.hasCache {
		return h.cacheBuf, nil
	}
	return io.ReadAll(h.Stream)
}

func (h *ReadableStream) CacheBuffer() ([]byte, error) {
	if h.hasCache {
		return h.cacheBuf, nil
	}

	buf, err := io.ReadAll(h.Stream)
	if err != nil {
		return nil, err
	}
	h.cacheBuf = buf
	h.hasCache = true
	h.Stream = neweofByteReadCloser(buf)
	return h.cacheBuf, nil
}

// If the stream has a duplicated/shadow string cache, then return it otherwise
// not
func (h *ReadableStream) TryCacheBuffer() ([]byte, bool) {
	if h.hasCache {
		return h.cacheBuf, true
	}
	return nil, false
}

func (h *ReadableStream) HasCache() bool {
	return h.hasCache
}

func (h *ReadableStream) ToStream() io.ReadCloser {
	return h.Stream
}

func (h *ReadableStream) SetStream(stream io.ReadCloser) {
	h.cacheBuf = []byte{}
	h.hasCache = false
	h.Stream = stream
}

func (h *ReadableStream) SetBuffer(b []byte) {
	h.cacheBuf = b
	h.hasCache = true
	h.Stream = neweofByteReadCloser(b)
}

func (h *ReadableStream) SetString(data string) {
	h.cacheBuf = []byte(data)
	h.hasCache = true
	h.Stream = neweofByteReadCloserFromString(data)
}

func (h *ReadableStream) Index(name pl.Val) (pl.Val, error) {
	if name.Type != pl.ValStr {
		return pl.NewValNull(), fmt.Errorf("invalid index, .readablestream field name must be string")
	}
	switch name.String() {
	case "hasCache":
		return pl.NewValBool(h.HasCache()), nil
	case "close":
		return pl.NewValBool(h.IsClose()), nil
	case "byteLength":
		if l := h.ByteLength(); l >= 0 {
			return pl.NewValInt(l), nil
		} else {
			return pl.NewValNull(), nil
		}
	default:
		return pl.NewValNull(), fmt.Errorf("invalid index, unknown name %s", name.String())
	}
}

func (h *ReadableStream) IndexSet(_ pl.Val, _ pl.Val) error {
	return fmt.Errorf(".readablestream does not support index set")
}

func (h *ReadableStream) Dot(name string) (pl.Val, error) {
	return h.Index(pl.NewValStr(name))
}

func (h *ReadableStream) DotSet(name string, value pl.Val) error {
	return fmt.Errorf(".readablestream does not support dot set")
}

func (h *ReadableStream) ToString() (string, error) {
	b, err := h.CacheBuffer()
	if err != nil {
		return "", err
	} else {
		return string(b), nil
	}
}

func (h *ReadableStream) ToJSON() (pl.Val, error) {
	b, ok := h.TryCacheBuffer()
	return pl.MarshalVal(
		map[string]interface{}{
			"content":  b,
			"hasCache": ok,
			"close":    h.IsClose(),
		},
	)
}

var (
	methodProtoReadableStreamCacheString    = pl.MustNewFuncProto(".readablestream.cacheString", "%0")
	methodProtoReadableStreamTryCacheString = pl.MustNewFuncProto(".readablestream.tryCacheString", "%0")
	methodProtoReadableStreamAsString       = pl.MustNewFuncProto(".readablestream.string", "%0")
	methodProtoReadableStreamClose          = pl.MustNewFuncProto(".readablestream.close", "%0")
)

func (h *ReadableStream) Method(name string, arg []pl.Val) (pl.Val, error) {
	switch name {
	case "cacheString":
		if _, err := methodProtoReadableStreamCacheString.Check(arg); err != nil {
			return pl.NewValNull(), err
		}
		if b, err := h.CacheBuffer(); err != nil {
			return pl.NewValNull(), err
		} else {
			return pl.NewValStr(string(b)), nil
		}
	case "tryCacheString":
		if _, err := methodProtoReadableStreamTryCacheString.Check(arg); err != nil {
			return pl.NewValNull(), err
		}
		b, ok := h.TryCacheBuffer()
		if !ok {
			return pl.NewValNull(), nil
		} else {
			return pl.NewValStr(string(b)), nil
		}
	// consume APIs
	case "string":
		if _, err := methodProtoReadableStreamAsString.Check(arg); err != nil {
			return pl.NewValNull(), err
		}
		s, err := h.ConsumeAsString()
		if err != nil {
			return pl.NewValNull(), err
		} else {
			return pl.NewValStr(s), nil
		}
	case "close":
		if _, err := methodProtoReadableStreamClose.Check(arg); err != nil {
			return pl.NewValNull(), err
		}
		if err := h.Close(); err != nil {
			return pl.NewValNull(), err
		} else {
			return pl.NewValNull(), nil
		}
	}
	return pl.NewValNull(), fmt.Errorf("method: .readablestream:%s is unknown", name)
}

func (h *ReadableStream) Info() string {
	return fmt.Sprintf(".readablestream[cache=%t;close=%t]", h.HasCache(), h.IsClose())
}

func (h *ReadableStream) IsThreadSafe() bool {
	return false
}

func (h *ReadableStream) Id() string {
	return ReadableStreamTypeId
}

func (h *ReadableStream) NewIterator() (pl.Iter, error) {
	return nil, fmt.Errorf(".readablestream does not support iterator")
}

func NewReadableStreamValFromStream(stream io.ReadCloser) pl.Val {
	x := NewReadableStreamFromStream(stream)
	return pl.NewValUsr(x)
}

func NewReadableStreamValFromString(data string) pl.Val {
	x := NewReadableStreamFromString(data)
	return pl.NewValUsr(x)
}

func NewReadableStreamValFromBuffer(data []byte) pl.Val {
	x := NewReadableStreamFromBuffer(data)
	return pl.NewValUsr(x)
}
