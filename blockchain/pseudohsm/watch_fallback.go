// +build ios linux,arm64 !darwin,!freebsd,!linux,!netbsd,!solaris,!windows

// This is the fallback implementation of directory watching.
// It is used on unsupported platforms.

package pseudohsm

type watcher struct{ running bool }

func newWatcher(*keyCache) *watcher { return new(watcher) }
func (*watcher) start()             {}
func (*watcher) close()             {}
