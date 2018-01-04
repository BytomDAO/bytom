package browser

import (
	"github.com/toqueteos/webbrowser"
)

// Open opens browser
func Open(url string) error {
	return webbrowser.Open(url)
}
