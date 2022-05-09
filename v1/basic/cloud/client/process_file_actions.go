package client

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/lazybark/go-helpers/hasher"
	"github.com/lazycloud-app/go-filesync/v1/basic/proto"
	"gorm.io/gorm"
)

func (c *Client) ProcessObjectRemoved(event proto.SyncEvent) (sResp ParseError) {
	dat, err := os.Stat(c.FW.UnEscapeAddress(filepath.Join(event.Path, event.Name)))
	if err != nil {
		sResp.Err = true
		sResp.Text = fmt.Errorf("object reading failed '%s': %w", c.FW.UnEscapeAddress(filepath.Join(event.Path, event.Name)), err)
		sResp.Type = proto.ErrInternal
		return
	}

	if dat.ModTime().After(event.NewUpdatedAt) {
		sResp.Err = true
		sResp.Text = fmt.Errorf("have newer version of '%s'", c.FW.UnEscapeAddress(filepath.Join(event.Path, event.Name)))
		sResp.Type = proto.ErrHaveNewerVersion
		return
	}

	var file proto.File
	var folder proto.Folder

	if event.ObjectType == proto.ObjectDir {

		err = c.DB.Where("name = ? and path = ?", event.Name, event.Path).First(&folder).Error
		if err != nil && err != gorm.ErrRecordNotFound {
			sResp.Err = true
			sResp.Text = fmt.Errorf("object DB reading failed '%s': %w", c.FW.UnEscapeAddress(filepath.Join(event.Path, event.Name)), err)
			sResp.Type = proto.ErrInternal
			return
		}
		err = c.DB.Delete(&folder).Error
		if err != nil {
			sResp.Err = true
			sResp.Text = fmt.Errorf("object DB deleting failed '%s': %w", c.FW.UnEscapeAddress(filepath.Join(event.Path, event.Name)), err)
			sResp.Type = proto.ErrInternal
			return
		}
		// Manually delete all files connected to this dir
		err = c.DB.Where("path = ?", c.FW.EscapeAddress(event.Name)).Delete(&file).Error
		if err != nil {
			sResp.Err = true
			sResp.Text = fmt.Errorf("object DB deleting related files failed '%s': %w", c.FW.UnEscapeAddress(filepath.Join(event.Path, event.Name)), err)
			sResp.Type = proto.ErrInternal
			return
		}
		err = os.RemoveAll(c.FW.UnEscapeAddress(filepath.Join(event.Path, event.Name)))
		if err != nil {
			sResp.Err = true
			sResp.Text = fmt.Errorf("error removing '%s': %w", c.FW.UnEscapeAddress(filepath.Join(event.Path, event.Name)), err)
			sResp.Type = proto.ErrInternal
			return
		}
	} else if event.ObjectType == proto.ObjectFile {
		err := c.DB.Where("name = ? and path = ?", event.Name, event.Path).First(&file).Error
		if err != nil && err != gorm.ErrRecordNotFound {
			sResp.Err = true
			sResp.Text = fmt.Errorf("object DB reading failed '%s': %w", c.FW.UnEscapeAddress(filepath.Join(event.Path, event.Name)), err)
			sResp.Type = proto.ErrInternal
			return
		}
		err = c.DB.Delete(&file).Error
		if err != nil {
			sResp.Err = true
			sResp.Text = fmt.Errorf("object DB deleting failed '%s': %w", c.FW.UnEscapeAddress(filepath.Join(event.Path, event.Name)), err)
			sResp.Type = proto.ErrInternal
			return
		}
		err = os.RemoveAll(c.FW.UnEscapeAddress(filepath.Join(event.Path, event.Name)))
		if err != nil {
			sResp.Err = true
			sResp.Text = fmt.Errorf("[RemoveAll] error removing '%s': %w", c.FW.UnEscapeAddress(filepath.Join(event.Path, event.Name)), err)
			sResp.Type = proto.ErrInternal
			return
		}
	}
	return
}

func (c *Client) ProcessObjectCreated(event proto.SyncEvent) (sResp ParseError) {
	if event.ObjectType == proto.ObjectDir {
		address := c.FW.UnEscapeAddress(filepath.Join(event.Path, event.Name))
		if err := os.MkdirAll(c.FW.UnEscapeAddress(filepath.Join(event.Path, event.Name)), os.ModePerm); err != nil {
			sResp.Err = true
			sResp.Text = fmt.Errorf("[ProcessObjectCreated] error making path '%s': %w", c.FW.UnEscapeAddress(filepath.Join(event.Path, event.Name)), err)
			sResp.Type = proto.ErrInternal
			return
		}
		err := os.Chtimes(c.FW.UnEscapeAddress(filepath.Join(event.Path, event.Name)), event.NewUpdatedAt, event.NewUpdatedAt)
		if err != nil {
			sResp.Err = true
			sResp.Text = fmt.Errorf("[ProcessObjectCreated] error changing times '%s': %w", c.FW.UnEscapeAddress(filepath.Join(event.Path, event.Name)), err)
			sResp.Type = proto.ErrInternal
			return
		}

		// Scan dir
		_, _, err, errs := c.FW.ProcessDirectory(address)
		if err != nil {
			sResp.Err = true
			sResp.Text = fmt.Errorf("[ProcessObjectCreated] processing dir '%s': %w", c.FW.UnEscapeAddress(filepath.Join(event.Path, event.Name)), err)
			sResp.Type = proto.ErrInternal
			return
		}
		if len(errs) > 0 {
			for _, v := range errs {
				fmt.Println(v)
			}
		}

		dat, err := os.Stat(c.FW.UnEscapeAddress(filepath.Join(event.Path, event.Name)))
		if err != nil {
			sResp.Err = true
			sResp.Text = fmt.Errorf("[ProcessObjectCreated] object reading failed '%s': %w", c.FW.UnEscapeAddress(filepath.Join(event.Path, event.Name)), err)
			sResp.Type = proto.ErrInternal
			return
		}

		// Add object to DB
		err = c.FW.MakeDBRecord(dat, address)
		if err != nil && err.Error() != "UNIQUE constraint failed: name, path" {
			sResp.Err = true
			sResp.Text = fmt.Errorf("[ProcessObjectCreated] error making record '%s': %w", c.FW.UnEscapeAddress(filepath.Join(event.Path, event.Name)), err)
			sResp.Type = proto.ErrInternal
			return
		}
		/*****************************************************/
		/*****************************************************/
		/*****************************************************/
		/*****************************************************/
		/* And here we will ask server to send full list of directory files in case in was crated not empty*/
		/*****************************************************/
		/*****************************************************/
		/*****************************************************/

	} else if event.ObjectType == proto.ObjectFile {
		fileToGet := proto.GetFile{
			Name:      event.Name,
			Path:      event.Path,
			Hash:      event.Hash,
			UpdatedAt: event.NewUpdatedAt,
		}
		c.FileGetter <- fileToGet
	}
	return
}

func (c *Client) ProcessObjectUpdated(event proto.SyncEvent) (sResp ParseError) {
	if event.ObjectType == proto.ObjectDir {
		// Dir only needs updating it's update time
		// As far as files events will be sent separately
		err := os.Chtimes(c.FW.UnEscapeAddress(filepath.Join(event.Path, event.Name)), event.NewUpdatedAt, event.NewUpdatedAt)
		if err != nil {
			sResp.Err = true
			sResp.Text = fmt.Errorf("[ProcessObjectUpdated] error changing times '%s': %w", c.FW.UnEscapeAddress(filepath.Join(event.Path, event.Name)), err)
			sResp.Type = proto.ErrInternal
			return
		}
		// Update data in DB
		var folder proto.Folder
		if err := c.DB.Where("name = ? and path = ?", event.Name, c.FW.EscapeAddress(event.Path)).First(&folder).Error; err != nil {
			sResp.Err = true
			sResp.Text = fmt.Errorf("[ProcessObjectUpdated] folder reading failed '%s': %w", c.FW.UnEscapeAddress(filepath.Join(event.Path, event.Name)), err)
			sResp.Type = proto.ErrInternal
			return
		} else {
			folder.FSUpdatedAt = event.NewUpdatedAt
			if err := c.DB.Table("folders").Save(&folder).Error; err != nil && err != gorm.ErrEmptySlice {
				sResp.Err = true
				sResp.Text = fmt.Errorf("[ProcessObjectUpdated] folder saving failed '%s': %w", c.FW.UnEscapeAddress(filepath.Join(event.Path, event.Name)), err)
				sResp.Type = proto.ErrInternal
				return
			}
		}
	} else {
		// File should be donwloaded only in case different hash value
		var file proto.File
		if err := c.DB.Where("name = ? and path = ?", event.Name, c.FW.EscapeAddress(event.Path)).First(&file).Error; err != nil {
			sResp.Err = true
			sResp.Text = fmt.Errorf("[ProcessObjectUpdated]File reading failed '%s': %w", c.FW.UnEscapeAddress(filepath.Join(event.Path, event.Name)), err)
			sResp.Type = proto.ErrInternal
			return
		} else {
			// Update data in DB
			hash := ""
			hash, err := hasher.HashFilePath(c.FW.UnEscapeAddress(filepath.Join(event.Path, event.Name)), hasher.SHA256, 8192)
			if err != nil {
				sResp.Err = true
				sResp.Text = fmt.Errorf("[ProcessObjectUpdated] error hashing file '%s': %w", c.FW.UnEscapeAddress(filepath.Join(event.Path, event.Name)), err)
				sResp.Type = proto.ErrInternal
				return
			}
			if hash == event.Hash {
				file.Hash = hash
				file.FSUpdatedAt = event.NewUpdatedAt
				if err := c.DB.Table("files").Save(&file).Error; err != nil && err != gorm.ErrEmptySlice {
					c.Logger.Error("Dir saving failed: ", err)
				}
			} else {
				fileToGet := proto.GetFile{
					Name:      event.Name,
					Path:      event.Path,
					Hash:      event.Hash,
					UpdatedAt: event.NewUpdatedAt,
				}
				c.FileGetter <- fileToGet
			}
		}
	}
	return
}
