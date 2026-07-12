package config

type Config struct {
	Server ServerConfig
	DB     DBConfig
	Redis  RedisConfig
}

type ServerConfig struct {
	Port int
}

type DBConfig struct {
	Host     string
	Port     int
	DBname   string
	Passwd   string
	Username string
}

type RedisConfig struct {
	Host   string
	Port   int
	Passwd string
	DB     int
}
