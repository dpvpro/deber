# deber

![](https://github.com/dpvpro/deber/workflows/Tests/badge.svg)
[![GoDoc](https://godoc.org/github.com/dpvpro/deber?status.svg)](https://godoc.org/github.com/dpvpro/deber)
[![go report card](https://goreportcard.com/badge/github.com/dpvpro/deber)](https://goreportcard.com/report/github.com/dpvpro/deber)

### `Debian` **+** `Docker` **=** `deber`

Utility made with simplicity in mind to provide
an easy way for building Debian packages in
Docker containers.

## Screencast

[![asciicast](https://asciinema.org/a/H2bjgbvzYnFNZLvEZruztIdnZ.svg)](https://asciinema.org/a/H2bjgbvzYnFNZLvEZruztIdnZ)

## Features

- Build packages for Debian and Ubuntu
- Use official Debian and Ubuntu images from DockerHub
- Automatically determine if target distribution is Ubuntu or Debian
  by querying DockerHub API
- Skip already ran steps (not every one)
- Install extra local packages in container
- Plays nice with `gbp-buildpackage`
- Easy local package dependency resolve
- Don't clutter your parent directories with `.deb`, `.dsc` and company
- Every successfully built package goes to local repo automatically
  so you can easily build another package that depends on previous one
- Ability to provide custom `dpkg-buildpackage` and `lintian` options
- Packages downloaded by apt are stored in temporary directory,
  to avoid repetitive unnecessary network load
- Option to drop into interactive bash shell session in container,
  for debugging or other purposes
- Use network in build process or not
- Automatically rebuilds image if old enough

## Installation

**Source (latest master)**

```bash
go install github.com/dpvpro/deber@latest
```

## Usage

I recommend to use deber with gbp if possible, but it will work just fine
as a standalone builder, like sbuild or pbuilder.

Let's assume that you are in directory with already debianized source, have
orig upstream tarball in parent directory and you want to build a package.
Just run:

```bash
deber
```

or if you use gbp and have `builder = deber` in `gbp.conf`

```bash
gbp buildpackage
```

If you run it first time, it will build Docker image and then proceed to build
your package.

To make use of packages from archive to build another package, specify desired directories with built artifacts and `deber` will take them to consideration when installing dependencies:

```bash
deber -p ~/deber/unstable/pkg1/1.0.0-1 -p ~/deber/unstable/pkg2/2.0.0-2
```

## FAQ

**Ok, everything went well, but... where is my `.deb`?!**

The location for all build outputs defaults to `$TMP/deber/packages`.

**Where is build directory located?**

`$TMP/deber/builddir/$CONTAINER`

**Where is apt's cache directory located?**

`$TMP/deber/cachedir/$IMAGE`

**How images built by deber are named?**

`deber:$DIST`

**I have already built image but it is building again?!**

Probably because it is 14 days old and deber decided to
update it.

**How to build a package for different distributions?**

Make a new entry with desired target distribution in `debian/changelog`
and run `deber`.

Or specify the desired distribution with `--distribution` option.

## CONTRIBUTING

I appreciate any contributions, so feel free to do so!
