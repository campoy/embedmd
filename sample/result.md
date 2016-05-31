# A hello world in Go

Go is very simple, here you can see a whole "hello, world" program.

[embedmd]:# (hello.go)
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

[embedmd]:# (hello.go /package .*\n/)
```go
package main
```

Followed by an `import` statement:

[embedmd]:# (hello.go /import/ /\)/)
```go
import (
	"fmt"
	"time"
)
```

You can also see how to get the current time:

[embedmd]:# (hello.go /time\.[^)]*\)/)
```go
time.Now()```


You can also have some extra code independent from `embedmd`

```python
        print 'hello'
```
