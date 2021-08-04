# dupers

Dupers is the blazing-fast file duplicate checker and filename search.

- Uses SHA-256 checksums in the fast and simple Bolt key/value database store.
- Automate the deletion of duplicates.
- Multithreaded file reads and scans.
- Instant filename and directory path searches from the database.
- Automated database maintenance with optional user tools.

## Downloads

<small>dupers is a standalone (portable) terminal application and doesn't require installation.</small>

- [Windows](https://github.com/bengarrett/dupers/releases/latest/download/dupers_Windows_Intel.zip); [or legacy 32-bit](https://github.com/bengarrett/dupers/releases/latest/download/dupers_Windows_32bit.zip)
- [macOS](https://github.com/bengarrett/dupers/releases/latest/download/dupers_macOS_Intel.tar.gz
); [or for the Apple M chip](https://github.com/bengarrett/dupers/releases/latest/download/dupers_macOS_M-series.tar.gz
)
- [FreeBSD](https://github.com/bengarrett/dupers/releases/latest/download/dupers_FreeBSD_Intel.tar.gz
)
- [Linux](https://github.com/bengarrett/dupers/releases/latest/download/dupers_Linux_Intel.tar.gz
)

#### Packages

##### [APK (Alpine package)](https://github.com/bengarrett/dupers/releases/latest/download/dupers.apk)
```sh
wget https://github.com/bengarrett/dupers/releases/latest/download/dupers.apk
apk add dupers.apk
```

##### [DEB (Debian package)](https://github.com/bengarrett/dupers/releases/latest/download/dupers.deb)
```sh
wget https://github.com/bengarrett/dupers/releases/latest/download/dupers.deb
dpkg -i dupers.deb
```

##### [RPM (Red Hat package)](https://github.com/bengarrett/dupers/releases/latest/download/dupers.rpm)
```sh
wget https://github.com/bengarrett/dupers/releases/latest/download/dupers.rpm
rpm -i dupers.rpm
```

##### Windows [Scoop](https://scoop.sh/)
```sh
scoop bucket add dupers https://github.com/bengarrett/dupers.git
scoop install dupers
```

## Usage

TODO screenshots

## Example usage
#### Dupe check

Run a check of the files in Downloads on the collection of text files.

```sh
# Windows
duper dupe C:\Users\Me\Downloads D:\textfiles

# Linux, macOS
duper dupe ~/Downloads ~/textfiles
```

#### Dupe check multiple locations

Run a check of the files in Downloads on collections of text files and images.

```sh
# Windows
duper dupe C:\Users\Me\Downloads D:\textfiles D:\photos

# Linux, macOS
duper dupe ~/Downloads ~/Textfiles ~/Pictures
```

#### Search for a filename
```sh
# Search the database for ZIP files
duper -name search .zip

# Search the database for photos containing 2010 in their file or directory names
duper search '2010' D:\photos
```

## Performance

Due to the nature of duplicate file checking there are a number of unrelated issues that can greatly effect the performance of duper.

#### Live statuses and command flags.

The terminal and command prompt apps are not designed for the rapid display of live text and can introduce a huge performance hit to dupers when processing large tasks. An easy fix is to use the `-quiet` flag with the duper commands.

```sh
# dupe command using quiet fast mode, takes less than a second
duper -quiet -fast dupe C:\Users\Me\Downloads D:\textfiles
# Scanned 191842 files, taking 901ms
```

```sh
# dupe command taking 46 seconds
duper dupe C:\Users\Me\Downloads D:\textfiles
# Checking 51179 of 387859 items...
# Scanned 191842 files, taking 46.3s

# dupe command using quiet mode, taking 15 seconds
duper -quiet dupe C:\Users\Me\Downloads D:\textfiles
# Scanned 191842 files, taking 15.0s

# dupe command using fast mode, taking 9 seconds
duper -quiet dupe C:\Users\Me\Downloads D:\textfiles
# Scanned 191842 files, taking 9.0s
```

#### Windows real-time protection.

It is highly encouraged that Windows users temporarily disable **Virus & threat protection, Real-time protection**, or [create **Windows Security Exclusion**s](https://support.microsoft.com/en-us/windows/add-an-exclusion-to-windows-security-811816c0-4dfd-af4a-47e4-c301afe13b26) for the folders to be scanned before running `dupers`. Otherwise, Defender will lock, open and read every file dupers scans, greatly slowing down the process.

#### Hardware

to-do

## Limitations

#### Identical files within a bucket are not saved to the database.

Dupers uses the SHA-256 file checksums as unique keys and each key's value only holds a single path location. This means both the `dupe` and `search` commands will not return all the possible locations of identical files in a bucket, as only one unique file is ever stored.


#### Windows directory paths

Go, the language dupers is created on treats the backslash `\` as an escape character. This unfortunately means Windows users cannot use tailing backslashes with directories.

##### Good
```ps
dupers dupe "C:\Users\Ben\Some directory"
```

##### Incorrect
```ps
dupers dupe "C:\Users\Ben\Some directory\"
```

## Troubleshoot

#### Windows

> `Not enough memory resources are available to process this command.`

This is a misleading generic Windows error that occurs when interacting with the database.
There is no guaranteed fix but try rebooting or running this command:

```ps
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
