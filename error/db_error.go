package newerror

import (
	"errors"
	"fmt"
	"net/http"

	"github.com/go-sql-driver/mysql"
	"gorm.io/gorm"
)

func IsMysqlError(result *gorm.DB) error {
	err := result.Error
	if err != nil {
		if isContext, err2 := IsContextError(err); isContext {
			return err2
		}
		if errors.Is(err, gorm.ErrDuplicatedKey) {
			return MakeError(CodeResourceDuplicate, fmt.Errorf(`The Key Already Set, Should Use "Update"`), LevelWarn).
				SetHttpError(http.StatusConflict, "The Key Already Exist")
		}
		if mysqlErr, ok := err.(*mysql.MySQLError); ok && mysqlErr.Number == 1062 {
			return MakeError(CodeResourceDuplicate, fmt.Errorf(`The Key Already Set, Should Use "Update"`), LevelWarn).
				SetHttpError(http.StatusConflict, "The Key Already Exist")
		}
		return MakeError(CodeDatabaseError, err, LevelError).
			SetHttpError(http.StatusInternalServerError, "Database Error")
	}
	return nil
}
