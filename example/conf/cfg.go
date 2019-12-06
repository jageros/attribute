package conf

import (
	"github.com/BurntSushi/toml"
	"log"
)

var cfg *Cfg

type Cfg struct {
	DBConfig *DatabaseCfg `toml:"database"`
}

// ================== DatabaseCfg ==================
type DatabaseCfg struct {
	Type            string `toml:"type"`
	Addr            string `toml:"addr"`
	User            string `toml:"user"`
	Password        string `toml:"password"`
	DBName          string `toml:"dbName"`
}

func (df *DatabaseCfg) GetType() string {
	return df.Type
}
func (df *DatabaseCfg) GetAddr() string {
	return df.Addr
}
func (df *DatabaseCfg) GetDB() string {
	return df.DBName
}
func (df *DatabaseCfg) GetUser() string {
	return df.User
}
func (df *DatabaseCfg) GetPassword() string {
	return df.Password
}

// ===================================================

func init() {
	var config *Cfg
	if _, err := toml.DecodeFile("conf/config.toml", &config); err != nil {
		log.Printf("Load toml config err=%+v\n", err)
	}
	cfg = config
}

func GetDBCfg() *DatabaseCfg {
	return cfg.DBConfig
}
