package orm

import (
	"time"
)

type NodeLiveness struct {
	ID         uint64 `gorm:"primary_key"`
	NodeID     uint16
	PingTimes  uint64
	PongTimes  uint64
	BestHeight uint64
	CreatedAt  time.Time
	UpdatedAt  time.Time

	Node *Node `gorm:"foreignkey:NodeID"`
}
