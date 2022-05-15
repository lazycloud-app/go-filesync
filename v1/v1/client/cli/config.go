package cli

import (
	"fmt"
	"log"
	"os"
	"strconv"

	"github.com/lazycloud-app/go-filesync/config"
	"github.com/lazycloud-app/go-filesync/helpers"
	"github.com/lazycloud-app/go-fsp-proto/ver"
	"golang.org/x/term"
)

var (
	//Current version of the client app
	version = ver.AppVersion{
		Major:        1,
		MLabel:       "test client",
		Minor:        0,
		Patch:        0,
		ReleaseName:  "testing release",
		ReleaseDate:  "2022.05.05",
		ReleaseSatus: "pre-alpha",
	}
)

//Config loads config from file to memory
func (c *Client) Config(conf string) error {
	decodedConf, err := config.Decode_v1(conf)
	if err != nil {
		return fmt.Errorf("[Config] error decoding config file -> %w", err)
	}

	c.conf = *decodedConf

	return nil
}

//ConfigNew creates new empty config file
func (c *Client) ConfigNew(conf string) error {
	confFile, err := os.Create(conf)
	if err != nil {
		return fmt.Errorf("[ConfigNew] error creating config file -> %w", err)
	}

	if err = config.Create_v1(confFile); err != nil {
		return fmt.Errorf("[ConfigNew] making new config -> %w", err)
	}

	confFile.Close()

	return nil
}

//ConfigOpenCLI calls to Config and in case of error allows user to pick another file or create new config via CLI (using ConfigNew)
func (c *Client) ConfigOpenCLI(conf string) {
	for {
		err := c.Config(conf)
		if err == nil {
			break
		}

		fmt.Printf("Can not open config file -> %s\nActions:\n[1] Pick another file\n[2] Create new config\n[3] Exit\n", err)
		ans, err := strconv.Atoi(helpers.ScanInputToString())
		if err != nil {
			log.Fatalf("Error parsing user input: %s", err)
		}

		if ans == 1 {
			for {
				fmt.Println("Provide existing config file or 'exit' to break:")
				conf = helpers.ScanInputToString()
				if conf == "" {
					fmt.Println("Config file name must not be empty")
					continue
				} else if conf == "exit" {
					fmt.Println("Exiting")
					os.Exit(1)
				}
				break
			}
		} else if ans == 2 {
			for {
				var name string
				for {
					fmt.Println("Enter new config file name or 'exit' to break:")
					name = helpers.ScanInputToString()
					if name == "" {
						fmt.Println("Config file name must not be empty")
						continue
					} else if name == "exit" {
						os.Exit(1)
					}
					break
				}
				if err := c.ConfigNew(name); err != nil {
					fmt.Printf("Error making new config file: %s\n", err)
					continue
				}
				fmt.Printf("\nCreated!\nNow you need to fill all fields in your new config at %s.\nIf you wish to hide sensitive data (such as password), you may leave some fields blank.\nIn that case you will be asked to enter values on startup.\n", name)
				os.Exit(1)
			}
		} else if ans == 3 {
			os.Exit(1)
		}
	}

	if c.conf.LOGIN == "" {
		for {
			fmt.Println("Enter login or 'exit' to break:")
			login := helpers.ScanInputToString()
			if login == "" {
				fmt.Println("Login must not be empty")
				continue
			} else if login == "exit" {
				os.Exit(1)
			}
			c.conf.LOGIN = login
			break
		}
	}
	if c.conf.PASSWORD == "" {
		for {
			fmt.Println("Enter password or 'exit' to break:")
			password, err := term.ReadPassword(int(os.Stdin.Fd()))
			if err != nil {
				fmt.Printf("Error reading password %s", err)
				continue
			}
			passwordStr := string(password)

			if passwordStr == "" {
				fmt.Println("Password must not be empty")
				continue
			} else if passwordStr == "exit" {
				os.Exit(1)
			}
			c.conf.PASSWORD = passwordStr
			break
		}
	}
	if c.conf.SERVER_ADDRESS == "" {
		for {
			fmt.Println("Enter server address or 'exit' to break:")
			address := helpers.ScanInputToString()
			if address == "" {
				fmt.Println("Server address must not be empty")
				continue
			} else if address == "exit" {
				os.Exit(1)
			}
			c.conf.SERVER_ADDRESS = address
			break
		}
	}
	if c.conf.SERVER_PORT == 0 {
		for {
			fmt.Println("Enter server port or '0' to break:")
			port, err := strconv.Atoi(helpers.ScanInputToString())
			if err != nil {
				fmt.Printf("Server port must not be empty %s\n", err)
				continue
			}
			if port == 0 {
				os.Exit(1)
			}
			c.conf.SERVER_PORT = port
			break
		}
	}
	if c.conf.FILE_SYSTEM_ROOT_PATH == "" {
		for {
			fmt.Println("Provide path to filesystem root or 'exit' to break:")
			path := helpers.ScanInputToString()
			if path == "" {
				fmt.Println("Path must not be empty")
				continue
			} else if path == "exit" {
				os.Exit(1)
			}
			c.conf.FILE_SYSTEM_ROOT_PATH = path
			break
		}
	}
	if c.conf.DB_FILE_NAME == "" {
		c.conf.DB_FILE_NAME = "client.db"
	}
	if c.conf.DIR_CACHE == "" {
		c.conf.DIR_CACHE = "cache"
	}
	if c.conf.DIR_LOGS == "" {
		c.conf.DB_FILE_NAME = "logs"
	}
}
