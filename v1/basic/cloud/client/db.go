package client

import "github.com/lazycloud-app/go-filesync/v1/basic/proto"

func (c *Client) InitDB() (err error) {
	err = c.DB.Migrator().DropTable(&proto.File{}, &proto.Folder{})
	if err != nil {
		return err
	}
	err = c.DB.AutoMigrate(&proto.File{}, &proto.Folder{})

	return
}
