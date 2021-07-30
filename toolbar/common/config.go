package common

type MySQLConfig struct {
	Connection MySQLConnection `json:"connection"`
	LogMode    bool            `json:"log_mode"`
}

type MySQLConnection struct {
	Host     string `json:"host"`
	Port     uint   `json:"port"`
	Username string `json:"username"`
	Password string `json:"password"`
	DbName   string `json:"database"`
}
