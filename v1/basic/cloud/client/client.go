package client

import (
	"bufio"
	"crypto/tls"
	"crypto/x509"
	"errors"
	"fmt"
	"io"
	"log"
	"net"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/fsnotify/fsnotify"
	"github.com/lazybark/go-pretty-code/logs"
	"github.com/lazycloud-app/go-filesync/helpers"
	"github.com/lazycloud-app/go-filesync/v1/basic/fs"
	"github.com/lazycloud-app/go-filesync/v1/basic/messenger"
	"github.com/lazycloud-app/go-filesync/v1/basic/proto"
	"github.com/spf13/viper"
	"go.uber.org/zap"
)

// NewSyncServer creates a sync server instance
func NewSyncClient() *Client {
	return new(Client)
}

func (c *Client) LogfileName() string {
	return fmt.Sprintf("go_filesync_%v-%v-%v_%v-%v-%v.log", c.TimeStart.Year(), c.TimeStart.Month(), c.TimeStart.Day(), c.TimeStart.Hour(), c.TimeStart.Minute(), c.TimeStart.Second())
}

func (c *Client) Start() {
	c.TimeStart = time.Now()
	c.AppVersion = proto.ClientVersion
	c.ActionsBuffer = make(map[string][]BufferedAction)
	c.Stat = new(Statistics)
	c.wg = new(sync.WaitGroup)

	err := c.LoadConfig(".")
	if err != nil {
		log.Fatal("Error getting config: ", err)
	}

	c.Logger, err = logs.Double(filepath.Join(c.Config.LogDirMain, c.LogfileName()), false, zap.InfoLevel)
	if err != nil {
		log.Fatal("Error getting logger: ", err)
	}
	c.Logger.Info(fmt.Sprintf("App Version: %s", c.AppVersion))

	// Connect DB
	c.DB, err = helpers.OpenSQLite(c.Config.SQLiteDBName)
	if err != nil {
		c.Logger.Fatal(fmt.Errorf("db add failed: %w", err))
	}

	// Connect FS watcher
	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		c.Logger.Fatal(fmt.Errorf("new watcher failed: %w", err))
	}

	// Force rescan filesystem and flush old DB-records
	c.InitDB()
	if err != nil {
		c.Logger.Fatal(fmt.Errorf("error flushing DB: %w", err))
	}

	c.syncEventsChan = make(chan fs.EventArray)
	c.syncErrChan = make(chan error)
	c.EventsChannel = make(chan proto.FSEvent)

	go c.NotificationPublisher()
	c.fp = fs.NewProcessor(c.Config.FileSystemRootPath, watcher, c.DB)
	go c.fp.FilesystemWatcherRoutine(c.syncEventsChan, c.syncErrChan)

	c.processRoot()

	// New filesystem worker
	//c.FW = fs.NewProcessor(c.Config.FileSystemRootPath, watcher, c.DB)

	// Watch root dir
	c.Logger.InfoGreen("Starting watcher")
	//go c.FilesystemWatcherRoutine()

	// Process and watch all subdirs
	//rpStart := time.Now()
	/*files, dirs, err, errs := c.FW.ProcessDirectory(c.Config.FileSystemRootPath)
	if err != nil {
		c.Logger.Fatal("Error processing FS: ", err)
	}
	if len(errs) > 0 {
		text := ""
		for _, e := range errs {
			text += e.Error() + "\n"
		}
		c.Logger.Warn("Errors in processing filesystem: \n" + text)
	}*/
	//c.Logger.InfoGreen("Root directory processed. Total time: ", time.Since(c.TimeStart))
	//c.Logger.InfoGreen(fmt.Sprintf("Root directory processed. Total %d files in %d directories. Time: %v", files, dirs, time.Since(rpStart)))

	c.Logger.InfoGreen("Starting client")

	c.Sync()
	c.wg.Wait()
}

func (c *Client) InitTLSConnection() (conn *tls.Conn, err error) {
	cert, err := os.ReadFile(c.Config.ServerCert)
	if err != nil {
		return conn, fmt.Errorf("[os.ReadFile] unable to read file -> %w", err)
	}
	certPool := x509.NewCertPool()
	if ok := certPool.AppendCertsFromPEM(cert); !ok {
		return conn, fmt.Errorf("[x509.NewCertPool] unable to parse cert from %s -> %w", c.Config.ServerCert, err)
	}
	config := &tls.Config{RootCAs: certPool}
	conn, err = tls.DialWithDialer(&net.Dialer{Timeout: 30 * time.Second}, "tcp", fmt.Sprintf("%s:%d", c.Config.ServerAddress, c.Config.ServerPort), config)
	if err != nil {
		return conn, fmt.Errorf("[tls.Dial] unable to dial to %s:%d -> %w", c.Config.ServerAddress, c.Config.ServerPort, err)
	}

	return
}

func (c *Client) LoadConfig(path string) (err error) {
	viper.AddConfigPath(path)
	viper.SetConfigName("config")
	viper.SetConfigType("json")

	viper.AutomaticEnv()

	err = viper.ReadInConfig()
	if err != nil {
		return
	}

	err = viper.Unmarshal(&c.Config)
	if err != nil {
		return
	}

	return
}

/*
// FilesystemWatcherRoutine tracks changes in every folder in root dir
func (c *Client) FilesystemWatcherRoutine() {
	done := make(chan bool)
	go func() {
		defer close(done)

		for {
			select {
			case event, ok := <-c.FW.Watcher.Events:
				if !ok {
					return
				}

				if len(c.ActionsBuffer[event.Name]) > 0 {
					var skip bool
					for n, a := range c.ActionsBuffer[event.Name] {
						if a.Action == event.Op && !a.Skipped {
							skip = true
							fmt.Println("ACTION SKIPPED FOR ", event.Name)
							c.ActionsBufferMutex.Lock()
							c.ActionsBuffer[event.Name][n].Skipped = true
							c.ActionsBufferMutex.Unlock()
						}
					}
					if skip {

						continue
					}

				}

				select {
				case c.fsEventsChan <- event:

					c.Logger.Info(fmt.Sprintf("%s %s", event.Name, event.Op))

				default:

				}

				// Pause before processing actions to make sure that target isn't locked
				// If file hashing still produces errors (target busy) - increase pause time
				time.Sleep(100 * time.Millisecond)

			case err, ok := <-c.FW.Watcher.Errors:
				if !ok {
					return
				}
				c.Logger.Error("FS watcher error: ", err)
			}
		}
	}()

	err := c.FW.Watcher.Add(c.Config.FileSystemRootPath)
	if err != nil {
		c.Logger.Fatal("FS watcher add failed: ", err)
	}
	c.Logger.Info(fmt.Sprintf("%s added to watcher", c.Config.FileSystemRootPath))
	<-done
}*/

func (c *Client) GetFile(fileToGet *proto.GetFile, conn *tls.Conn, m *messenger.Messenger) {
	c.wg.Add(1)
	defer c.wg.Done()
	rec := m.Recieved()

	// Telling that we want to get the file
	err := m.SendMessage(fileToGet, proto.MessageGetFile, conn)
	if err != nil {
		c.Logger.Error("[GetFile] sending file request error -> ", err)
		return
	}
	c.Stat.BytesSent += m.SBytes()

	// Await answer with file
	for {
		netData, err := bufio.NewReader(conn).ReadBytes(proto.Terminator)
		if err != nil {
			// If connection closed - break the cycle
			if errors.As(err, &io.ErrClosedPipe) {
				c.Logger.Info("[GetFile] server has closed the connection")
				return
			}
			c.Logger.Error(fmt.Sprintf("[GetFile] ReadBytes -> error reading data: %v", err))
			continue
		}

		err = m.ParseRecieved(&netData)
		if err != nil {
			c.Logger.Error(fmt.Sprintf("[GetFile] ParseRecieved -> broken message: %v", err))
			continue
		}

		fullPath := new(string)
		var updatedAt *time.Time

		if rec.Type == proto.MessageSendFile {
			file, err := m.ParseFileData()
			if err != nil {
				c.Logger.Error(fmt.Sprintf("[GetFile] ParseFileData -> broken message: %v", err))
				continue
			}

			*fullPath = c.fp.UnEscapeAddress(filepath.Join(file.Path, file.Name))
			updatedAt = &file.FSUpdatedAt
			var lastByte int

			// We need to ignore file creation to avoid fs-watcher process event that we have created
			c.AddActionToBuffer(*fullPath, fsnotify.Create)

			cacheAddress, e := c.CreateFileInCache()
			if e != nil {
				c.Logger.Error(err)
				return
			}

			for {
				netData2, err := bufio.NewReader(conn).ReadBytes(proto.Terminator)
				if err != nil {

					// If connection closed - break the cycle
					if errors.As(err, &io.ErrClosedPipe) {
						c.Logger.Info("[GetFile] server has closed the connection")
						//HERE WE CAN STORE LAST DOWNLOADED BYTE AND RESUME DOWNLOAD IN CASE HASH IS THE SAME
						fmt.Println("last byte is", lastByte)
						return
					}
					c.Logger.Error(fmt.Sprintf("(%v)[ReadBytes] - error reading data: %v", conn.RemoteAddr(), err))
					continue
				}

				err = m.ParseRecieved(&netData2)
				if err != nil {
					c.Logger.Error(fmt.Sprintf("(%v)[message.Parse] - broken message: %v", conn.RemoteAddr(), err))
					continue
				}

				if rec.Type == proto.MessageFileParts {
					theFile, err := os.OpenFile(cacheAddress, os.O_APPEND, 0666)
					if err != nil {
						fmt.Println("err opening", err)
						return
					}
					n, err := theFile.Write(rec.Payload)
					if err != nil {
						fmt.Println("Errpr", err)
						return
					}

					// here in case of error we can process last
					lastByte += n

					theFile.Close()

					continue

				} else if rec.Type == proto.MessageFileEnd {

					fmt.Println("Got file end")

					err = os.Chtimes(cacheAddress, *updatedAt, *updatedAt)
					if err != nil {
						fmt.Println(err)
					}

					// Move the file to right position
					err := os.Rename(cacheAddress, *fullPath)
					if err != nil {
						c.Logger.Error("moving failed: ", *fullPath, err)
					}

					_, err = os.Stat(*fullPath)
					if err != nil {
						c.Logger.Error("Object reading failed: ", *fullPath, err)
						continue
					}

					file := fs.File{
						Hash:        fileToGet.Hash,
						Name:        fileToGet.Name,
						Path:        fileToGet.Path,
						Size:        int64(lastByte),
						Ext:         filepath.Ext(fileToGet.Name),
						FSUpdatedAt: fileToGet.UpdatedAt,
					}

					// Add object to DB
					err = c.fp.RecordFile(file)
					if err != nil && err.Error() != "UNIQUE constraint failed: files.name, files.path" {
						c.Logger.Error(fmt.Sprintf("Error making record for %s: ", *fullPath), err)
					}
					break
				}
			}

			return
		} else if rec.Type == proto.MessageError {
			c.Logger.Error("[GetFile] %v", c.ProcessErrorPayload(rec.Payload))
			return
		} else {
			c.Logger.Error("[GetFile] server responded with and unexpected message type %s", rec.Type)
			return
		}
	}
}

func (c *Client) NotificationPublisher() {
	for {
		select {
		case e := <-c.syncEventsChan:
			//If conversion result says that this action type is not for exchanging with other parties - just skip
			//It is useful in case action series (ex.: FS_RENAME->FS_CREATE)
			if e.Proto.Action == proto.FS_NO_ACTION {
				continue
			}
			//Notify clients about the Event
			fmt.Println(e)
			c.EventsChannel <- e.Proto
			/*for _, c := range s.pool.pool {
				if !c.syncActive || c.uid != e.FS.Owner {
					continue
				}
				c.eventsChan <- e.Proto
			}*/
		case e := <-c.syncErrChan:
			c.Logger.Error(fmt.Errorf("error in filesystem watcher: %w", e))
		}
	}
}

func (c *Client) processRoot() {
	c.Logger.InfoGreen("Processing root directory")
	rpStart := time.Now()
	files, dirs, err, errs := c.fp.ProcessDirectory(c.Config.FileSystemRootPath)
	if err != nil {
		c.Logger.Error(fmt.Errorf("error processing directory: %w", err))
	}
	if len(errs) > 0 {
		text := ""
		for _, e := range errs {
			text += e.Error() + "\n"
		}
		c.Logger.Warn("Errors in processing filesystem: \n" + text)
	}
	c.Logger.InfoGreen(fmt.Sprintf("Root directory processed. Total %d files in %d directories. Time: %v", files, dirs, time.Since(rpStart)))
	c.Logger.InfoGreen("Starting server")
}
