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
		for _, c := range s.activeConnections {
			// Count stat by session key in the connection
			fmt.Println(c)
		}
	}
}
