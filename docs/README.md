# dupers

Dupers is the blazing-fast file duplicate checker and filename search.

- Dupers uses SHA-256 checksums in the fast and straightforward key/value database store.
- Automate the deletion of duplicates.
- Multithreaded file reads and scans.
- Instant filename and directory path search from the database.
- Automated database maintenance with optional user tools.
- **TODO:** Import and export database stores as [CSV text](https://en.wikipedia.org/wiki/Comma-separated_values) for sharing.

## Downloads

<small>Dupers is a standalone (portable) terminal program and doesn't require installation.</small>

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
```ps
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
duper search "2010" D:\photos
```

## Performance

Due to the nature of duplicate file checking, several unrelated issues can significantly affect duper's performance.

#### Hardware and the Operating System

With its constant opening and reading of files, hardware directly affects dupers' performance, both CPU used and the read/write speed of the hard drive. Fast multithreaded CPUs and SSD drives help here.

Recent changes to modern operating systems harm dupers. Nowadays, terminal and command prompt applications receive 25-35% of the available CPU resources.

You can improve this by adjusting the process priority of dupers in your operating system's activity/system processes tool, but it may not give the desired effect.

#### Live status and command flags

The terminal and command prompt apps are not designed for the rapid display of live text and can introduce a significant performance hit to dupers when processing large tasks. An easy fix is to use the `-quiet` flag with the duper commands.

###### Dupe command using *quiet* fast mode takes less than a second 😃
```ps
duper -quiet -fast dupe C:\Users\Me\Downloads D:\textfiles
# Scanned 191842 files, taking 901ms
```
###### Dupe command using the fast mode, taking 9 seconds
```ps
duper -fast dupe C:\Users\Me\Downloads D:\textfiles
# Scanned 191842 files, taking 9.0s
```
###### Dupe command using the *quiet* mode, taking 15 seconds
```ps
duper -quiet dupe C:\Users\Me\Downloads D:\textfiles
# Scanned 191842 files, taking 15.0s
```
###### Dupe command taking 46 seconds ☹️
```ps
duper dupe C:\Users\Me\Downloads D:\textfiles
# Checking 51179 of 387859 items...
# Scanned 191842 files, taking 46.3s
```

## Limitations

#### Identical files within a bucket are not saved to the database

Dupers uses the SHA-256 file checksums as unique keys and each key's value only holds a single path location. This means both the `dupe` and `search` commands will not return all the possible locations of identical files in a bucket, as only one unique file is ever stored.


#### Windows Command Prompt directory paths

Windows Command Prompt (cmd.exe) users cannot use tailing backslashes with quoted directories. Other terminal apps such as [Windows Terminal](https://www.microsoft.com/en-au/p/windows-terminal/9n0dx20hk701) do not suffer this issue.

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
