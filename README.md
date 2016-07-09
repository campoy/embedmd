[![Build Status](https://travis-ci.org/campoy/embedmd.svg)](https://travis-ci.org/campoy/embedmd)

# embedmd

Are you tired of copy pasting your code into your `README.md` file, just to
forget about it later on and have unsynced copies? Or even worse, code
that does not even compile?

Then `embedmd` is for you!

`embedmd` embeds files or fractions of files into markdown files. It does
so by searching `embedmd` commands, which are a subset of the markdown
syntax for comments. This means they are invisible when markdown is
rendered, so they can be kept in the file as pointers to the origin of
the embedded text.

The command receives a list of markdown files, if none is given it reads
from the standard input.

The format of an `embedmd` command is:

```markdown
[embedmd]:# (filename language /start regexp/ /end regexp/)
```

The embedded code will be extracted from the file filename, starting at the
first line that matches `/start regexp/` and finishing at the first line
matching `/end regexp/`.

Ommiting the the second regular expression will embed only the piece of text
that matches `/regexp/`:

```markdown
[embedmd]:# (filename language /regexp/)
```

To embed the whole line matching a regular expression you can use:

```markdown
[embedmd]:# (filename language /.*regexp.*/)
```

If you want to embed from a point to the end you should use:

```markdown
[embedmd]:# (filename language /start regexp/ $)
```

Finally you can embed a whole file by omitting both regular expressions:

```markdown
[embedmd]:# (filename language)
```

You can ommit the language in any of the previous commands, and the extension
of the file will be used for the snippet syntax highlighting.

Note that while this works Go files, since the file extension `.go` matches the
name of the language `go`, this will fail with other fileslike `.md` whose
language name is `markdown`.

```markdown
[embedmd]:# (file.ext)
```

## Installation

`embedmd` is written in Go, so if you have Go installed (you can do so
by following [these instructions](golang.org/doc/install.html)) you can
install it with go get:

```
go get github.com/campoy/embedmd
```

This will download the code, compile it, and leave an `embedmd` binary
in `$GOPATH/bin`.

Eventually, and if there's enough interest, I will provide binaries for
every OS and architecture out there ... _eventually_.

## Usage:

Given the two files in [sample](sample):

*hello.go:*

[embedmd]:# (sample/hello.go)
```go
// Copyright 2016 Google Inc. All rights reserved.
// Use of this source code is governed by the Apache 2.0
// license that can be found in the LICENSE file.

package main

import (
	"fmt"
	"time"
)

func main() {
	fmt.Println("Hello, there, it is", time.Now())
}
```

*docs.md:*

[embedmd]:# (sample/docs.md markdown /./ /embedmd.*time.*/)
```markdown
# A hello world in Go

Go is very simple, here you can see a whole "hello, world" program.

[embedmd]:# (hello.go)

You always start with a `package` statement like:

[embedmd]:# (hello.go /package.*/)

Followed by an `import` statement:

[embedmd]:# (hello.go /import/ /\)/)

You can also see how to get the current time:

[embedmd]:# (hello.go /time\.[^)]*\)/)
```

Executing `embedmd -w docs.md` will modify `docs.md`
and add the corresponding code snippets, as shown in
[sample/result.md](sample/result.md).

### Disclaimer

This is not an official Google product (experimental or otherwise), it is just
code that happens to be owned by Google.
