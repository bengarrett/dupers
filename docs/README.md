# dupers

[![PkgGoDev](https://pkg.go.dev/badge/github.com/bengarrett/dupers)](https://pkg.go.dev/github.com/bengarrett/dupers)
[![Go Report Card](https://goreportcard.com/badge/github.com/bengarrett/dupers)](https://goreportcard.com/report/github.com/bengarrett/dupers)
[![GitHub release (latest SemVer)](https://img.shields.io/github/v/release/bengarrett/dupers)](https://github.com/bengarrett/dupers/releases)
![GitHub](https://img.shields.io/github/license/bengarrett/dupers?style=flat)

Dupers is the blazing-fast file duplicate checker and filename search tool.

- Uses SHA-256 checksums stored in a fast key/value database for accurate duplicate detection
- Safe, automated duplicate deletion with user confirmation
- Multithreaded file processing for maximum performance
- Instant filename and directory path search from the database
- Automated database maintenance with optional user tools
- Cross-platform support (Windows, macOS, Linux)
- Import/export database stores as [CSV](https://en.wikipedia.org/wiki/Comma-separated_values) for sharing

## Downloads

Dupers is available as standalone portable binaries and system packages. No installation is required for the portable versions.

### Portable Binaries

**Windows:** [Download](https://github.com/bengarrett/dupers/releases/latest/download/dupers_windows_amd64.zip)

**Linux:** [Download](https://github.com/bengarrett/dupers/releases/latest/download/dupers_linux_amd64.tar.gz)

**macOS:** [Download](https://github.com/bengarrett/dupers/releases/latest/download/dupers_macOS_all.tar.gz)

Before use, macOS users will need to delete the 'quarantine' extended attribute that is applied to all 
program downloads that are not notarized by Apple for a fee.

```
$ xattr -d com.apple.quarantine namzd
```

#### Homebrew

macOS and Linux users can install via Homebrew:

```bash
brew tap bengarrett/dupers https://github.com/bengarrett/dupers
brew install bengarrett/dupers/dupers
```

Update to the latest version with:

```bash
brew upgrade bengarrett/dupers/dupers
```

### Linux Packages

##### [Ubuntu/Debian (.deb)](https://github.com/bengarrett/dupers/releases/latest/download/dupers_amd64.deb)
```sh
dpk -i dupers_amd64.deb
```

##### [Fedora (.rpm)](https://github.com/bengarrett/dupers/releases/latest/download/dupers_amd64.rpm)
```sh
rpm -i dupers_amd64.rpm
```

##### [Alpine Linux (.apk)](https://github.com/bengarrett/dupers/releases/latest/download/dupers_amd64.apk)
```sh
apk add dupers_amd64.apk
```

##### [Arch Linux (.zst)](https://github.com/bengarrett/dupers/releases/latest/download/dupers_amd64.pkg.tar.zst)
```sh
pacman -U dupers_amd64.pkg.tar.zst
```

## Quick Start

Get started with dupers in just a few commands:

```sh
# Windows users will use backslashes: dupers up ~\Documents

# Add your main directories to the database (buckets)
dupers up ~/Documents
dupers up ~/Downloads
dupers up /path/to/your/files

# Find duplicate files
dupers dupe ~/Pictures ~/Documents

# Search for files by name
dupers search "project"

# View database information
dupers database
```

## Example usage

#### Dupe check

Run a check to find duplicate photos.

```sh
dupers dupe ~/photos # Windows example: ~\photos or C:\photos

# dupers        the program name
# dupe          the command to run
# ~/photos     the path containing a collection of files (a bucket)
```

Run a check to see if the photo exists within the photo collection.

```sh
dupers dupe photo.jpg ~/photos # Windows example: ~\photos or C:\photos

# photo.jpg     the new file to check
# ~/photos      the path containing a collection of files (a bucket)
```

Run a check of the files in Downloads against the collection of stored files.

```sh
dupers dupe ~/Downloads ~/storage # Windows example: ~\Downloads C:\storage

# ~/Downloads  the path containing new files to check
# ~/storage    the path containing a collection of files (a bucket)
```

#### Dupe check multiple locations

Run a check of the files in Downloads against the collections of documents, music and images.

```sh
dupers dupe ~/Downloads ~/documents ~/images ~/music

# Windows example: dupers dupe ~\Downloads ~\Documents D:\images E:\music

# ~/Downloads  the path containing new files to check
# ~/documents  a path containing a collection of files (a bucket)
# ~/images     another path containing a collection of files (another bucket)
# ~/music      another path containing a collection of files (and another bucket)
```

#### Search for a filename

Search the database for ZIP files.

Note: options such as `-name` always go before the command.

```sh
dupers -name search .zip

# dupers     the program name
# -name      an option, to search only for filenames
# search     the command to run
# .zip       the search expression
```


Search the database for photos containing '2010' in their file or directory names.

```sh
dupers search "2010" ~/photos # Windows example: D:\photos

# dupers     the program name
# search     the command to run
# "2010"     the search expression
# ~/photos   the path containing a collection of files (a bucket)
```

## Performance

Due to the nature of duplicate file checking, hardware and operating systems do affect performance.

#### The fast flag

When running dupe checking, a `-fast` flag can significantly improve performance when dealing with extensive file collections. It does this by only running duplicate checks against the database and completely ignoring the files residing on the host system.

###### Dupe command on a large collection using fast mode takes less than a second 😃
```sh
dupers -fast dupe C:\Users\Me\Downloads D:\textfiles
# Scanned 191842 files, taking 901ms
```

###### Dupe command on a large collection normally taking 46 seconds ☹️
```sh
dupers dupe C:\Users\Me\Downloads D:\textfiles
# Checking 51179 of 387859 items...
# Scanned 191842 files, taking 46.3s
```

## Limitations

#### Multiple identical files

Both the `dupe` and `search` commands __only show the first matching file__. Dupers uses the SHA-256 file checksums as unique keys, and each key value holds a single location path.

#### Command Prompt directories

The legacy Windows Command Prompt (`cmd.exe`) cannot use trailing backslashes with quoted directories. Windows Terminal does not suffer this issue.

##### ✔️ Good
```ps
dupers dupe "C:\Users\Ben\Some directory"
```

##### ❌ Incorrect
```ps
dupers dupe "C:\Users\Ben\Some directory\"
```

## Troubleshoot

#### Windows

> `Not enough memory resources are available to process this command.`

This is a misleading generic Windows error that occurs when interacting with the database.
There is no guaranteed fix but try rebooting or running this command:

```sh
# In an administrator console or administrator command prompt.
sfc /scannow
```

## Build

[Go](https://golang.org/doc/install) supports dozens of architectures and operating systems letting dupers to [be built for most platforms](https://golang.org/doc/install/source#environment).

```sh
# clone this repo
git clone git@github.com:bengarrett/dupers.git

# access the repo
cd dupers

# target and build the app for the host system
go build

# target and build for OpenBSD
env GOOS=openbsd GOARCH=amd64 go build
```
