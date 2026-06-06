package config

type Config struct {
	HTTPAddr string
}

func Load() Config {
	return Config{
		HTTPAddr: ":8080",
	}
}
