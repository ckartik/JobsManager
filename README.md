# Design Doc

## Library

The core of the library revolves around a single object called `JobsManager`.
``` go
type JobsManager struct {
	Jobs sync.Map
	Output sync.Map
}
```