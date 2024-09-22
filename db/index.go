package db

import (
	"fmt"

	"xorm.io/xorm"
)

var Engine *xorm.Engine

func init() {
	var err error
	Engine, err = xorm.NewEngine("mysql", "root:root@/gonet?charset=utf8")
	if err != nil {
		fmt.Println(err.Error())
	}
}
