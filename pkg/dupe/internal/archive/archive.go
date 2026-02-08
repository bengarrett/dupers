// © Ben Garrett https://github.com/bengarrett/dupers

// Package archive provides archive file handling and MIME type detection.
package archive

import (
	"errors"
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

	Ext7z = ".7z" // Ext7z is the 7-Zip file extension.
)

var (
	ErrFilename = errors.New("filename is not a supported archive")
	ErrType     = errors.New("archiver type is unsupported")
)

// Extension finds either a compressed file extension or mime type and returns its match.
// The function uses deterministic lookup with priority given to longer, more specific extensions.
func Extension(find string) string {
	if find == "" {
		return ""
	}

	lfind := strings.ToLower(find)

	// Direct map lookup for extensions (deterministic)
	if ext, ok := extensionToMIME[lfind]; ok {
		return ext
	}

	// Check for extension without dot prefix (e.g., "zip" -> ".zip")
	if !strings.HasPrefix(find, ".") {
		if _, ok := extensionToMIME["."+lfind]; ok {
			// For filename without dot prefix, return the extension itself, not the MIME type
			return "." + lfind
		}
	}

	// Direct map lookup for MIME types (deterministic) - lowest priority
	if ext, ok := mimeToExtension[lfind]; ok {
		return ext
	}

	return ""
}

// Predefined maps for deterministic lookup.
// These are effectively constant lookup tables and are safe as package-level variables
// since they are only used for read operations and never modified after initialization.
//
//nolint:gochecknoglobals // These are internal package constants used for lookup tables
var (
	extensionToMIME = map[string]string{
		Ext7z:      Mime7z,
		".bz2":     MimeBZ2,
		".gz":      MimeGZ,
		".lz4":     MimeLZ4,
		".rar":     MimeRAR,
		".sz":      MimeSnap,
		".tar":     MimeTar,
		".tar.br":  MimeTar,
		".tbr":     MimeTar,
		".tar.gz":  MimeTar,
		".tgz":     MimeTar,
		".tar.bz2": MimeTar,
		".tbz2":    MimeTar,
		".tar.xz":  MimeTar,
		".txz":     MimeTar,
		".tar.lz4": MimeTar,
		".tlz4":    MimeTar,
		".tar.sz":  MimeTar,
		".tsz":     MimeTar,
		".tar.zst": MimeTar,
		".xz":      MimeXZ,
		".zip":     MimeZip,
		".zst":     MimeZ,
	}

	mimeToExtension = map[string]string{
		Mime7z:   Ext7z,
		MimeBZ2:  ".bz2",
		MimeGZ:   ".gz",
		MimeLZ4:  ".lz4",
		MimeRAR:  ".rar",
		MimeSnap: ".sz",
		MimeTar:  ".tar",
		MimeXZ:   ".xz",
		MimeZip:  ".zip",
		MimeZ:    ".zst",
	}
)

// MIME returns the application MIME type of a filename based on its extension.
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

// ReadMIME opens and reads the named file and returns its compressed application MIME type.
// If the compression format is unsupported or unknown an error is returned.
func ReadMIME(name string) (string, error) {
	name = filepath.Clean(name)
	f, err := os.Open(name)
	if err != nil {
		return "", err
	}
	defer func() {
		_ = f.Close()
	}()
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
