package models

import (
	"errors"
	"fmt"

	"github.com/VividCortex/mysqlerr"
	"github.com/go-sql-driver/mysql"
	"gorm.io/gorm"
)

func NotFound(err error) bool {
	return errors.Is(err, gorm.ErrRecordNotFound)
}

func GetErrMessage(err error) string {
	me := &mysql.MySQLError{}
	if !errors.As(err, &me) {
		return err.Error()
	}
	switch me.Number {
	case mysqlerr.ER_DUP_ENTRY:
		return fmt.Sprintf("存在重名对象(code=%v)", me.Number)
	case mysqlerr.ER_DATA_TOO_LONG:
		return fmt.Sprintf("数据超长(code=%v)", me.Number)
	case mysqlerr.ER_TRUNCATED_WRONG_VALUE:
		return fmt.Sprintf("日期格式错误(code=%v)", me.Number)
	case mysqlerr.ER_NO_REFERENCED_ROW_2:
		return fmt.Sprintf("系统错误(外键关联数据出错 code=%v)", me.Number)
	case mysqlerr.ER_ROW_IS_REFERENCED_2:
		return fmt.Sprintf("系统错误(外键关联数据错误 code=%v)", me.Number)
	default:
		return fmt.Sprintf("系统错误(code=%v, message=%v)!", me.Number, me.Message)
	}
}
