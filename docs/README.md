# dupers

Dupers is the blazing-fast file duplicate checker and filename search.

- TODO list features

## Downloads

<small>dupers is a standalone (portable) terminal application and doesn't require installation.</small>

- [Windows](https://github.com/bengarrett/dupers/releases/latest/download/dupers_Windows_Intel.zip)
- [macOS](https://github.com/bengarrett/dupers/releases/latest/download/dupers_macOS_Intel.tar.gz
), [or for the Apple M chip](https://github.com/bengarrett/dupers/releases/latest/download/dupers_macOS_M-series.tar.gz
)
- [FreeBSD](https://github.com/bengarrett/dupers/releases/latest/download/dupers_FreeBSD_Intel.tar.gz
)
- [Linux](https://github.com/bengarrett/dupers/releases/latest/download/dupers_Linux_Intel.tar.gz
)

### Packages

TODO

## Windows Performance

It is highly encouraged that Windows users temporarily disable **Virus & threat protection, Real-time protection**, or [create **Windows Security Exclusion**s](https://support.microsoft.com/en-us/windows/add-an-exclusion-to-windows-security-811816c0-4dfd-af4a-47e4-c301afe13b26) for the folders to be scanned before running `dupers`. Otherwise, the hit to performance is amazingly stark!

## Usage

TODO screenshots

## Example usage
#### Dupe check
```sh
# todo
```

#### Dupe check a second time
```sh
# todo, show the time taken difference
```

#### Search for a filename
```sh
# todo
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