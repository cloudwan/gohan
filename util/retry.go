package util

func Retry(f func() error, shouldRetry func(error) bool, attempts int) (err error) {
	for attempt := 0; attempt < attempts; attempt++ {
		err = f()
		if !shouldRetry(err) {
			break
		}
	}
	return
}
