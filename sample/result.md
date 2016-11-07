# A hello world in Go

Go is very simple, here you can see a whole "hello, world" program.

[embedmd]:# (hello.go)
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

We can try to embed a file from a directory.

[embedmd]:# (test/hello.go /func main/ $)
```go
func main() {
	fmt.Println("Hello, there, it is", time.Now())
}
```

You always start with a `package` statement like:

[embedmd]:# (hello.go /package.*/)
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
time.Now()
```

You can also have some extra code independent from `embedmd`

```python
print 'hello'
```

And why not include some file directly from GitHub?

[embedmd]:# (https://raw.githubusercontent.com/campoy/embedmd/master/sample/hello.go /func main/ $)
```go
func main() {
	fmt.Println("Hello, there, it is", time.Now())
}
```
