package config

type (
	ClientConfV1 struct {
		CONFIG_VER            int
		LOGIN                 string
		PASSWORD              string
		SERVER_CERT_FILE      string
		SERVER_ADDRESS        string
		SERVER_PORT           int
		DIR_LOGS              string
		DIR_CACHE             string
		FILE_SYSTEM_ROOT_PATH string
		DB_FILE_NAME          string
	}
)
