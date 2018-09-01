// +build darwin,!ios freebsd linux,!arm64 netbsd solaris windows

package pseudohsm

//"fmt"
//"github.com/rjeczalik/notify"
//"time"

type watcher struct {
	kc       *keyCache
	starting bool
	running  bool
	//ev       chan notify.EventInfo
	quit chan struct{}
}

func newWatcher(kc *keyCache) *watcher {
	return &watcher{
		kc: kc,
		//ev:   make(chan notify.EventInfo, 10),
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

	/*
		err := notify.Watch(w.kc.keydir, w.ev, notify.All)
		if err != nil {
			fmt.Printf("can't watch %s: %v", w.kc.keydir, err)
			return
		}
		defer notify.Stop(w.ev)
		fmt.Printf("now watching %s", w.kc.keydir)
		defer fmt.Printf("no longer watching %s", w.kc.keydir)

		w.kc.mu.Lock()
		w.running = true
		w.kc.mu.Unlock()

		// Wait for file system events and reload.
		// When an event occurs, the reload call is delayed a bit so that
		// multiple events arriving quickly only cause a single reload.
		var (
			debounce          = time.NewTimer(0)
			debounceDuration  = 500 * time.Millisecond
			inCycle, hadEvent bool
		)
		defer debounce.Stop()
		for {
			select {
			case <-w.quit:
				return
			case <-w.ev:
				if !inCycle {
					debounce.Reset(debounceDuration)
					inCycle = true
				} else {
					hadEvent = true
				}
			case <-debounce.C:
				w.kc.mu.Lock()
				w.kc.reload()
				w.kc.mu.Unlock()
				if hadEvent {
					debounce.Reset(debounceDuration)
					inCycle, hadEvent = true, false
				} else {
					inCycle, hadEvent = false, false
				}
			}
		}
	*/
}
