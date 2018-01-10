package performance

import (
	"testing"
)

func BechmarkRpc(b *testing.B) {
	b.StopTimer()
	b.StartTimer()
}
