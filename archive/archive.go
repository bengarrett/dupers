// © Ben Garrett https://github.com/bengarrett/dupers

// Package archive handles the metadata of known file archive types.
package archive

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/h2non/filetype"
	"github.com/mholt/archiver/v3"
)

const (
	Mime7z   = "application/x-7z-compressed"       // 7-Zip type.
	MimeAr   = "application/x-unix-archive"        // ar (Unix) type.
	MimeBZ2  = "application/x-bzip2"               // bzip2 type.
	MimeCab  = "application/vnd.ms-cab-compressed" // Microsoft cabinet type.
	MimeGZ   = "application/gzip"                  // GNU Zip type.
	MimeLZ   = "application/x-lzip"                // LZ type.
	MimeLZ4  = "application/x-lz4"                 // LZ4 type.
	MimeRAR  = "application/vnd.rar"               // RAR type.
	MimeSnap = "application/x-snappy-framed"       // Snappy type.
	MimeTar  = "application/x-tar"                 // Tape archive type.
	MimeX    = "application/x-compress"            // Huffman type.
	MimeXZ   = "application/x-xz"                  // XZ type.
	MimeZ    = "application/zstd"                  // Zstandard type.
	MimeZip  = "application/zip"                   // ZIP type.

	Ext7z = ".7z" // 7-Zip file extension.
)

var (
	ErrFilename = errors.New("filename is not a supported archive")
	ErrType     = errors.New("archiver type is unsupported")
)

// Extension returns the metadata type for a file archive based on the string value.
// If the string includes a file name or extension, a MIME datatype is returned.
// If string value matches a known MIME datatype, then the correlating file extension is returned.
func Extension(s string) string {
	f := strings.ToLower(s)
	for ext, mime := range exts() {
		if f == ext {
			return mime
		}
		if f == mime {
			return ext
		}
		if !strings.HasPrefix(s, ".") {
			if ext == fmt.Sprintf(".%s", f) {
				return ext
			}
		}
	}
	return ""
}

// exts returns known archive mime types, these refer to data types and do not contain encoding information.
// Known mime types are those detected by the h2non/filetype library.
func exts() map[string]string {
	return map[string]string{
		Ext7z:      Mime7z,
		".bz2":     MimeBZ2,
		".gz":      MimeGZ,   // gzip
		".lz4":     MimeLZ4,  // LZ4*
		".rar":     MimeRAR,  // rar
		".sz":      MimeSnap, // Snappy*
		".tar":     MimeTar,  // tar
		".tar.br":  MimeTar,  // tar + Brotli compression
		".tbr":     MimeTar,  //
		".tar.gz":  MimeTar,  // tar + gzip compression
		".tgz":     MimeTar,  //
		".tar.bz2": MimeTar,  // tar + bzip2 compression
		".tbz2":    MimeTar,  //
		".tar.xz":  MimeTar,  // tar + XZ compression
		".txz":     MimeTar,  //
		".tar.lz4": MimeTar,  // tar + LZ4 compression
		".tlz4":    MimeTar,  //
		".tar.sz":  MimeTar,  // tar + snappy compression
		".tsz":     MimeTar,  //
		".tar.zst": MimeTar,  // tar + Zstandard compression
		".xz":      MimeXZ,   // XZ Utils
		".zip":     MimeZip,
		".zst":     MimeZ, // Zstandard (zstd)*
	}
}

// MIME returns the application MIME type of the named file based on its extension.
func MIME(name string) string {
	ext := filepath.Ext(name)
	if ext == "" {
		return ""
	}
	if find := Extension(ext); find != "" {
		return find
	}
	return ""
}

// ReadMIME reads the named file and returns its compressed application MIME type.
// If the compression format is unsupported or unknown an error is returned.
func ReadMIME(name string) (string, error) {
	f, err := os.Open(name)
	if err != nil {
		return "", err
	}
	defer f.Close()
	kind, err := filetype.MatchReader(f)
	if err != nil {
		return "", err
	}
	switch kind.MIME.Value {
	case Mime7z, MimeBZ2, MimeGZ, MimeRAR, MimeTar, MimeZip:
		// supported archives
		return kind.MIME.Value, nil
	case MimeXZ, MimeCab, MimeX, MimeLZ, MimeAr:
		// unsupported archives by mholt/archiver/v3 v3.5.x
		return kind.MIME.Value, ErrFilename
	}
	// non-archives
	return "", ErrFilename
}

// Supported returns true when the archiver format structure is valid.
func Supported(f any) bool {
	switch f.(type) {
	case
		*archiver.Brotli,
		*archiver.Bz2,
		*archiver.Gz,
		*archiver.Lz4,
		*archiver.Rar,
		*archiver.Snappy,
		*archiver.Tar,
		*archiver.TarBrotli,
		*archiver.TarBz2,
		*archiver.TarGz,
		*archiver.TarLz4,
		*archiver.TarSz,
		*archiver.TarXz,
		*archiver.TarZstd,
		*archiver.Xz,
		*archiver.Zip,
		*archiver.Zstd:
		return true
	default:
		return false
	}
}
