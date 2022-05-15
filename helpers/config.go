package helpers

import "github.com/spf13/viper"

func LoadConfig(from string, to interface{}) (err error) {
	viper.AddConfigPath(from)
	viper.SetConfigName("config")
	viper.SetConfigType("json")

	viper.AutomaticEnv()

	err = viper.ReadInConfig()
	if err != nil {
		return
	}

	err = viper.Unmarshal(&to)
	if err != nil {
		return
	}

	return
}
