package db

import (
	"fmt"

	"xorm.io/xorm"
)

var Engine *xorm.Engine

func init() {
	var err error
	Engine, err = xorm.NewEngine("mysql", "root:p9Bhz2!69Q0M74@/simctl?charset=utf8")
	if err != nil {
		fmt.Println(err.Error())
	}
}
