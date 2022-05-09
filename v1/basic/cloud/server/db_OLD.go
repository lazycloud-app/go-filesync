package server

import (
	u "github.com/lazycloud-app/go-filesync/users"
	"github.com/lazycloud-app/go-filesync/v1/basic/proto"
)

func (s *Server) InitDB() (err error) {
	err = s.db.Migrator().DropTable(&proto.File{}, &proto.Folder{})
	if err != nil {
		return err
	}
	err = s.db.AutoMigrate(&u.User{}, &u.Client{}, &proto.File{}, &proto.Folder{}, &Statistics{}, &StatisticsBySession{})

	return
}
