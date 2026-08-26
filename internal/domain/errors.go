package domain

import "fmt"

var (
	ErrNotFound          = fmt.Errorf("记录不存在")
	ErrVersionConflict   = fmt.Errorf("任务版本冲突")
	ErrInvalidTransition = fmt.Errorf("不允许的状态迁移")
	ErrFrozen            = fmt.Errorf("数据已经冻结")
	ErrCredentialIssued  = fmt.Errorf("放行凭证已经签发")
)

type ValidationError struct{ Field, Message string }

func (e ValidationError) Error() string { return e.Field + ": " + e.Message }
func Required(field, value string) error {
	if value == "" {
		return ValidationError{field, "不能为空"}
	}
	return nil
}
