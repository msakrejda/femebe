package femebe

// Call given function repeatedly until an error is returned; then
// send the error on the given channel and exit
func errToChannel(fn func() error, ch chan <- error) {
	var err error
	for err = fn(); err == nil; err = fn() {}
	ch <- err
}

