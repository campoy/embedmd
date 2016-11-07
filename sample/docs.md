# A hello world in Go

Go is very simple, here you can see a whole "hello, world" program.

[embedmd]:# (hello.go)

We can try to embed a file from a directory.

[embedmd]:# (test/hello.go /func main/ $)

You always start with a `package` statement like:

[embedmd]:# (hello.go /package.*/)

Followed by an `import` statement:

[embedmd]:# (hello.go /import/ /\)/)

You can also see how to get the current time:

[embedmd]:# (hello.go /time\.[^)]*\)/)

You can also have some extra code independent from `embedmd`

```python
print 'hello'
```

And why not include some file directly from GitHub?

[embedmd]:# (https://raw.githubusercontent.com/campoy/embedmd/master/sample/hello.go /func main/ $)