// +build darwin,!ios freebsd linux,!arm64 netbsd solaris windows

package pseudohsm

type watcher struct {
	kc       *keyCache
	starting bool
	running  bool
	quit     chan struct{}
}

func newWatcher(kc *keyCache) *watcher {
	return &watcher{
		kc:   kc,
		quit: make(chan struct{}),
	}
}

// starts the watcher loop in the background.
// Start a watcher in the background if that's not already in progress.
// The caller must hold w.kc.mu.
func (w *watcher) start() {
	if w.starting || w.running {
		return
	}
	w.starting = true
	go w.loop()
}

func (w *watcher) close() {
	close(w.quit)
}

func (w *watcher) loop() {
	defer func() {
		w.kc.mu.Lock()
		w.running = false
		w.starting = false
		w.kc.mu.Unlock()
	}()
}
