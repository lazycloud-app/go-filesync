package client

import "github.com/lazycloud-app/go-filesync/v1/basic/fs"

func (c *Client) InitDB() (err error) {
	err = c.DB.Migrator().DropTable(&fs.File{}, &fs.Folder{})
	if err != nil {
		return err
	}
	err = c.DB.AutoMigrate(&fs.File{}, &fs.Folder{})

	return
}
