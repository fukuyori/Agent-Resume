package agent

import (
	"io"
	"os"
	"strings"

	"github.com/klauspost/compress/zstd"
)

// zstdJSONLSuffix is the extension Codex uses for compressed rollout files.
const zstdJSONLSuffix = ".jsonl.zst"

func isJSONLFile(name string) bool {
	return strings.HasSuffix(name, ".jsonl") || strings.HasSuffix(name, zstdJSONLSuffix)
}

func trimJSONLSuffix(name string) string {
	if strings.HasSuffix(name, zstdJSONLSuffix) {
		return strings.TrimSuffix(name, zstdJSONLSuffix)
	}
	return strings.TrimSuffix(name, ".jsonl")
}

type zstdReadCloser struct {
	*zstd.Decoder
	file *os.File
}

func (z *zstdReadCloser) Close() error {
	z.Decoder.Close()
	return z.file.Close()
}

// openJSONL opens a JSONL file, transparently decompressing .jsonl.zst files.
func openJSONL(path string) (io.ReadCloser, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	if !strings.HasSuffix(path, zstdJSONLSuffix) {
		return f, nil
	}
	dec, err := zstd.NewReader(f, zstd.WithDecoderConcurrency(1))
	if err != nil {
		f.Close()
		return nil, err
	}
	return &zstdReadCloser{Decoder: dec, file: f}, nil
}
