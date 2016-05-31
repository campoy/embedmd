# A hello world in Go

Go is very simple, here you can see a whole "hello, world" program.

[embed.md]:# (hello.go)
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

You always start with a `package` statement like:

[embed.md]:# (hello.go /pack/)
```go
package main```

Followed by an `import` statement:

[embed.md]:# (hello.go /import/ /\)/)
```go
import (
	"fmt"
	"time"
)```

And finall, the `main` function:

[embed.md]:# (hello.go /func main/ //)
```go
func main() {
	fmt.Println("Hello, there, it is", time.Now())
}
```
