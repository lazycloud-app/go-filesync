package server

import (
	"fmt"
	"time"
)

type (
	Statistics struct {
		ID                 uint `gorm:"primaryKey"`
		Date               time.Time
		TotalBytesRecieved int
		TotalBytesSent     int
		FilesSent          int
		FilesRecieved      int
		ActiveUsers        int
		ActiveSessions     int
		TimeInSync         int // Minutes in sync mode
	}
	StatisticsBySession struct {
		ID                 uint      `gorm:"primaryKey"`
		Uid                uint      `gorm:"uniqueIndex:sbd"`
		SessionKey         uint      `gorm:"uniqueIndex:sbd"`
		Date               time.Time `gorm:"uniqueIndex:sbd"`
		TotalBytesRecieved int
		TotalBytesSent     int
		FilesSent          int
		FilesRecieved      int
		TimeInSync         int // Minutes in sync mode
	}
)

func (s *Server) CountStats() {
	for {
		time.Sleep(5 * time.Minute)
		var activeConnectionsNumber int
		var activeSync int

		for _, c := range s.pool.pool {
			if c.active {
				activeConnectionsNumber++
			}
			if c.syncActive {
				activeSync++
			}
		}
		if s.config.LogStats {
			s.Info(fmt.Sprintf("Server stats: active users = 0, active connections = %d (in sync %d), data recieved = 0, data sent = 0, errors last 15 min / hour / 24 hours = 0/0/0, time online = %v", activeConnectionsNumber, activeSync, time.Since(s.timeStart)))
		}
	}
}
