# Named Mutex

Named Mutex is a simple package that provides a mutex map.

## Instalation

```sh
go get -u github.com/yudai/nmutex
```

## Example

```go
	m := nmutex.New()

	releaseABC := m.Lock("abc") // you can get a lock

	releaseXYZ := m.Lock("xyz") // you can get a lock with another name
	defer releaseXYZ()          // using defer to release locks is a best practice

	go func() {
		// this call is blocked until releaseABC() has been called
		releaseABC := m.Lock("abc")
		defer releaseABC()
	}()

	releaseABC() // release the first lock
	// the goroutine above gets the lock at this timing
```

When the lock for a name has been released and there is no other gouroutines waiting for the lock, the internal resource for the name is automatically released.
