package db

import (
	"fmt"
	"log"
	"simctl/config"

	"xorm.io/xorm"
)

var Engine *xorm.Engine

func init() {
	var err error
	Engine, err = xorm.NewEngine("mysql", fmt.Sprintf("root:%v@/simctl?charset=utf8", config.DbPassword))
	if err != nil {
		log.Println(err.Error())
	}
}
