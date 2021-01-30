This repository contains an implementation of the [PAR1 file
format](http://parchive.sourceforge.net/docs/specifications/parity-volume-spec-1.0/article-spec.html)
and the [PAR2 file
format](http://parchive.sourceforge.net/docs/specifications/parity-volume-spec/article-spec.html)
in Go, as well as a command-line application for manipulating PAR1 and
PAR2 files.

### Installation

To install the par command-line application:

```
go get -u github.com/akalin/gopar/cmd/par
```

or run

```
go build cmd/par/main.go
```

after a checkout of this repository.

### Usage

Create, verify and repair parity files. Run `par --help` for an overview of all options.

### libgopar

Use gopar in your own Go code. Feed a string array with the commandline arguments to Gopar():

```
import "github.com/akalin/gopar/libgopar"
my_args := []strings
libgopar.Gopar(my_args)
```

Alternatively, compile a C library:

```
go build -buildmode=c-shared -o libgopar.so ./libgopar/C.go
```

This library exposes the `gopar()` function.

### License

Use of this source code is governed by a BSD-style license that can be
found in the LICENSE file.
