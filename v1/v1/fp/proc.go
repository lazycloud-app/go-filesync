//WatchRoot calls to Watch using root directory as the argument
func (fp *FP) WatchRoot() {
	fp.Watch(fp.root)
}

//Watch sets internal watcher to monitor filesystem changes in d. Errors are returned via fp.errChan
func (fp *FP) Watch(d string) {
	if CheckEscaped(d) {
		d = fp.UnEscapeAddress(d)
	}
	err := fp.w.Add(d)
	if err != nil {
		fp.errChan <- fmt.Errorf("[Watch] watcher for '%s' failed: %w", d, err)
	}
}
