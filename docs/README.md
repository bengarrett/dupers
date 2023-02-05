# dupers

Dupers is the blazing-fast file duplicate checker and filename search tool.

- Dupers uses SHA-256 checksums in the fast and straightforward key/value database store.
- Automate the deletion of duplicates.
- Multithreaded file reads and scans.
- Instant filename and directory path search from the database.
- Automated database maintenance with optional user tools.
- Import and export database stores as [CSV text](https://en.wikipedia.org/wiki/Comma-separated_values) for sharing.

## Downloads

<small>Dupers is a standalone (portable) terminal program and doesn't require installation.</small>

- [Windows](https://github.com/bengarrett/dupers/releases/latest/download/dupers_windows_amd64.zip) or [a legacy 32-bit edition](https://github.com/bengarrett/dupers/releases/latest/download/dupers_windows_386.zip).
- [macOS](https://github.com/bengarrett/dupers/releases/latest/download/dupers_macOS_all.tar.gz
), [Linux](https://github.com/bengarrett/dupers/releases/latest/download/dupers_linux_amd64.tar.gz
) or [Linux for ARM CPUs](https://github.com/bengarrett/dupers/releases/latest/download/dupers_linux_arm64.tar.tar.gz
).

#### Packages

##### [Homebrew](https://brew.sh/) for macOS, Linux, on Intel and ARM
```sh
brew install bengarrett/tap/dupers
```

##### [Scoop](https://scoop.sh/) for Windows
```ps
scoop bucket add dupers https://github.com/bengarrett/dupers.git
scoop install dupers
```

##### [DEB (Debian package)](https://github.com/bengarrett/dupers/releases/latest/download/dupers_amd64.deb)
```sh
wget https://github.com/bengarrett/dupers/releases/latest/download/dupers_amd64.deb
dpkg -i dupers_amd64.deb
```

##### [RPM (Red Hat package)](https://github.com/bengarrett/dupers/releases/latest/download/dupers_amd64.rpm)
```sh
wget https://github.com/bengarrett/dupers/releases/latest/download/dupers_amd64.rpm
rpm -i dupers_amd64.rpm
```

##### [APK (Alpine package)](https://github.com/bengarrett/dupers/releases/latest/download/dupers_amd64.apk)
```sh
wget https://github.com/bengarrett/dupers/releases/latest/download/dupers_amd64.apk
apk add dupers_amd64.apk
```

##### [Arch Linux](https://github.com/bengarrett/dupers/releases/latest/download/dupers_amd64.pkg.tar.zst)
```sh
wget https://github.com/bengarrett/dupers/releases/latest/download/dupers_amd64.pkg.tar.zst
pacman -U dupers_amd64.pkg.tar.zst
```

## Usage

https://user-images.githubusercontent.com/513842/140050025-04adc6ad-f7a4-4680-b83f-3fa3016f1504.mp4

## Example usage

#### Dupe check

Run a check to find any duplicate photos.

```sh
# Windows
dupers dupe D:\photos

# Linux, macOS
dupers dupe ~/photos

# dupers        the program name
# dupe          the command to run
# D:\photos     the path containing a collection of files (a bucket)
```

Run a check to see if the photo exists within the photo collection.

```sh
# Windows
dupers dupe photo.jpg D:\photos

# Linux, macOS
dupers dupe photo.jpg ~/photos

# photo.jpg     the new file to check
# D:\photos     the path containing a collection of files (a bucket)
```

Run a check of the files in Downloads against the collection of storage files.

```sh
# Windows
dupers dupe C:\Downloads D:\Storage

# Linux, macOS
dupers dupe ~/Downloads ~/storage

# C:\Downloads  the path containing new files to check
# D:\Storage    the path containing a collection of files (a bucket)
```

#### Dupe check multiple locations

Run a check of the files in Downloads against the collections of documents, music and images.

```sh
# Windows
dupers dupe C:\Downloads D:\documents D:\images E:\music

# Linux, macOS
dupers dupe ~/Downloads ~/documents ~/images ~/music

# C:\Downloads  the path containing new files to check
# D:\documents  a path containing a collection of files (a bucket)
# D:\images     another path containing a collection of files (another bucket)
# D:\music      another path containing a collection of files (and another bucket)
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


Search the database for photos containing `2010` in their file or directory names.

```sh
dupers search "2010" D:\photos

# dupers     the program name
# search     the command to run
# "2010"     the search expression
# D:\photos  the path containing a collection of files (a bucket)
```

## Performance

Due to the nature of duplicate file checking, several issues can affect performance.

#### Hardware
With its constant opening and reading of files, hardware directly affects dupers' performance, CPU used and the read/write speed of the hard drive. Fast multithreaded CPUs and SSD drives help here.

#### Operating Systems

To restrict aggressive programs, terminal and command prompt applications only receive **25-35%** of the available CPU resources. You can improve this by adjusting the process priority of dupers in your operating system's activity or processes tool, but it may not give the desired effect.

#### Command flags

When running dupe checking, a `-fast` flag can significantly improve performance when dealing with extensive file collections. It does this by only running dupe checks against the database and completely ignoring the files residing on the host system.

###### Dupe command on a large collection using fast mode takes less than a second 😃
```ps
dupers -fast dupe C:\Users\Me\Downloads D:\textfiles
# Scanned 191842 files, taking 901ms
```

###### Dupe command on a large collection normally taking 46 seconds ☹️
```ps
dupers dupe C:\Users\Me\Downloads D:\textfiles
# Checking 51179 of 387859 items...
# Scanned 191842 files, taking 46.3s
```

## Limitations

#### Multiple identical files

Both the `dupe` and `search` commands only show the first unique file match. Dupers uses the SHA-256 file checksums as unique keys, and each key's value only holds a single path location.

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
