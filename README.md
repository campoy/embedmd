# embed.md

embed.md parses all the .md files in a given directory looking for embed.md
commands. For every one of the commands it extracts some code and embeds it
as markdown code right below the command.

The command format for embed.md follows the markdown comments syntax, which
makes it invisible while rendering. This also allows to keep a reference to
the origin of the embedded code, therefore providing a way to update the
embedded copy if the original file changes.

The format of an embed.md command is:

```markdown
        [embed.md]:# (filename language /start regexp/ /end regexp/)
```

The embedded code will be extracted from the file filename, starting at the
first line that matches /start regexp/ and finishing at the first line
matching /end regexp/.

Ommiting the the second regular expression will embed only the line that
matches /start regexp/:

```markdown
    [embed.md]:# (filename language /regexp/)
```

If you want to embed from a point to the end you should use:

```markdown
    [embed.md]:# (filename language /start regexp/ //)
```

Finally you can embed a whole file by omitting both regular expressions:

```markdown
    [embed.md]:# (filename language)
```

You can ommit the language in any of the previous commands, and the extension
of the file will be used for the snippet syntax highlighting. Note that while
this works Go files, it will fail with most language such as `.md` vs `markdown`.

```markdown
    [embed.md]:# (file.ext)
```

## Installation

`embed.md` is written in Go, so if you have it installed (you can do so
by following [these instructions](golang.org/doc/install.html)) simply
run:

```
    go get github.com/campoy/embed.md
```

This will download the code, compile it, and generate an `embed.md` binary
in `$GOPATH/bin`.

Eventually, and if there's enough interest, I will provide binaries for
every OS and architecture out there ... _eventually_.

## Usage:

Given the two files in [sample](sample):

*hello.go:*
[embed.md]:# (sample/hello.go)
```go
package main

import (
	"fmt"
	"time"
)

func main() {
	fmt.Println("Hello, there, it is", time.Now())
}
```

*docs.md*
[embed.md]:# (sample/docs.md markdown)
```markdown
# A hello world in Go

Go is very simple, here you can see a whole "hello, world" program.

[embed.md]:# (hello.go)

You always start with a `package` statement like:

[embed.md]:# (hello.go /pack/)

Followed by an `import` statement:

[embed.md]:# (hello.go /import/ /\)/)

And finall, the `main` function:

[embed.md]:# (hello.go /func main/ //)
```

Executing `embed.md` in the directory containing `docs.md` will modify `docs.md`
and add the corresponding code snippets, as shown in
[sample/restult.md](sample/restult.md).

### Disclaimer

This is not an official Google product (experimental or otherwise), it is just
code that happens to be owned by Google.