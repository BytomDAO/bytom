package orm

import (
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/bytom/bytom/toolbar/common"
	precogCommon "github.com/bytom/bytom/toolbar/precognitive/common"
)

type Node struct {
	ID                       uint16 `gorm:"primary_key"`
	Alias                    string
	Xpub                     string
	PublicKey                string
	IP                       string
	Port                     uint16
	BestHeight               uint64
	AvgRttMS                 sql.NullInt64
	LatestDailyUptimeMinutes uint64
	Status                   uint8
	CreatedAt                time.Time
	UpdatedAt                time.Time
}

func (n *Node) MarshalJSON() ([]byte, error) {
	status, ok := precogCommon.StatusLookupTable[n.Status]
	if !ok {
		return nil, errors.New("fail to look up status")
	}

	avgRttMS := uint64(0)
	if n.AvgRttMS.Valid {
		avgRttMS = uint64(n.AvgRttMS.Int64)
	}

	return json.Marshal(&struct {
		Alias                    string           `json:"alias"`
		PublicKey                string           `json:"publickey"`
		Address                  string           `json:"address"`
		BestHeight               uint64           `json:"best_height"`
		AvgRttMS                 uint64           `json:"avg_rtt_ms"`
		LatestDailyUptimeMinutes uint64           `json:"latest_daily_uptime_minutes"`
		Status                   string           `json:"status"`
		UpdatedAt                common.Timestamp `json:"updated_at"`
	}{
		Alias:                    n.Alias,
		PublicKey:                n.PublicKey,
		Address:                  fmt.Sprintf("%s:%d", n.IP, n.Port),
		BestHeight:               n.BestHeight,
		AvgRttMS:                 avgRttMS,
		LatestDailyUptimeMinutes: n.LatestDailyUptimeMinutes,
		Status:    status,
		UpdatedAt: common.Timestamp(n.UpdatedAt),
	})
}
