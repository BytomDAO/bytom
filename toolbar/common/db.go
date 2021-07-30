package common

import (
	"fmt"

	_ "github.com/go-sql-driver/mysql"
	"github.com/jinzhu/gorm"

	"github.com/bytom/bytom/errors"
)

func NewMySQLDB(cfg MySQLConfig) (*gorm.DB, error) {
	dsnTemplate := "%s:%s@tcp(%s:%d)/%s?charset=utf8&parseTime=true&loc=Local"
	dsn := fmt.Sprintf(dsnTemplate, cfg.Connection.Username, cfg.Connection.Password, cfg.Connection.Host, cfg.Connection.Port, cfg.Connection.DbName)
	db, err := gorm.Open("mysql", dsn)
	if err != nil {
		return nil, errors.Wrap(err, "open db cluster")
	}

	db.LogMode(cfg.LogMode)
	if err = db.DB().Ping(); err != nil {
		return nil, errors.Wrap(err, "ping db")
	}

	return db, nil
}
