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

- [Windows](https://github.com/bengarrett/dupers/releases/latest/download/dupers_Windows_Intel.zip) or [XP compatible 32-bit](https://github.com/bengarrett/dupers/releases/latest/download/dupers_Windows_32bit.zip)
- [macOS](https://github.com/bengarrett/dupers/releases/latest/download/dupers_macOS_all.tar.gz
)
- [Linux](https://github.com/bengarrett/dupers/releases/latest/download/dupers_Linux_Intel.tar.gz
) and [FreeBSD](https://github.com/bengarrett/dupers/releases/latest/download/dupers_FreeBSD_Intel.tar.gz
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

https://user-images.githubusercontent.com/513842/133888424-ae1d7872-c67e-47a3-b328-897f833e3aa5.mp4

## Example usage
#### Dupe check

Run a check of the files in Downloads on the collection of text files.

```sh
# Windows
dupers dupe C:\Downloads D:\textfiles

# Linux, macOS
dupers dupe ~/Downloads ~/textfiles

# dupers        the program name
# dupe          the command to run
# C:\Downloads  the path containing new files to check
# D:\textfiles  the path containing a collection of files (a bucket)
```

#### Dupe check multiple locations

Run a check of the files in Downloads on collections of text files and images.

```sh
# Windows
dupers dupe C:\Downloads D:\textfiles D:\photos

# Linux, macOS
dupers dupe ~/Downloads ~/Textfiles ~/Pictures

# dupers        the program name
# dupe          the command to run
# C:\Downloads  the path containing new files to check
# D:\textfiles  a path containing a collection of files (a bucket)
# D:\photos     another path containing a collection of files (a bucket)
```

#### Search for a filename
```sh
# Search the database for ZIP files
# Note: options such as -name always go before the command
dupers -name search .zip

# dupers     the program name
# -name      an option, to search only for filenames
# search     the command to run
# .zip       the search expression
```

```sh
# Search the database for photos containing 2010 in their file or directory names
dupers search "2010" D:\photos

# dupers     the program name
# search     the command to run
# "2010"     the search expression
# D:\photos  the path containing a collection of files (a bucket)
```

## Performance

Due to the nature of duplicate file checking, several issues can affect duper's performance.

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
