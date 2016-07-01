# A hello world in Go

Go is very simple, here you can see a whole "hello, world" program.

[embedmd]:# (hello.go)

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
