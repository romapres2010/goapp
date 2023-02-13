package errors

import (
    "fmt"
    "strconv"
    "strings"
    "sync/atomic"

    pkgerr "github.com/pkg/errors"

    _log "github.com/romapres2010/goapp/pkg/common/logger"
)

// ErrorMessage Сообщения
//
//easyjson:json
type ErrorMessage struct {
    Code string `json:"code"`
    Text string `json:"text"`
}

const ERR_UNDEFINED_ID = uint64(0)

var errorID uint64 // уникальный номер ошибки

// getNextErrorID - запросить номер следующей ошибки
func getNextErrorID() uint64 {
    return atomic.AddUint64(&errorID, 1)
}

// SetTypedErrorMessages - установить набор сообщений об ошибках
func SetTypedErrorMessages(errorMessage map[string]ErrorMessage) {
    globalTypedErrorMessages = errorMessage
}

// AppendTypedErrorMessages - добавить сообщения в набор сообщений об ошибках
func AppendTypedErrorMessages(errorMessage map[string]ErrorMessage) {
    for s, message := range errorMessage {
        globalTypedErrorMessages[s] = message
    }
}

// Error represent custom error
type Error struct {
    ID           uint64   `json:"id,omitempty"`            // уникальный номер ошибки
    ExternalId   uint64   `json:"external_id,omitempty"`   // внешний ID, который был передан при создании ошибки
    Code         string   `json:"code,omitempty"`          // код ошибки
    Message      string   `json:"message,omitempty"`       // текст ошибки
    Details      []string `json:"details,omitempty"`       // текст ошибки подробный
    Caller       string   `json:"caller,omitempty"`        // файл, строка и наименование метода в котором произошла ошибка
    Args         string   `json:"args,omitempty"`          // строка аргументов
    CauseMessage string   `json:"cause_message,omitempty"` // текст ошибки - причины
    CauseErr     error    `json:"cause_err,omitempty"`     // ошибка - причина
    Trace        string   `json:"trace,omitempty"`         // стек вызова
}

// Format output
//
//	%s    print the error code, message, arguments, and cause message.
//	%v    in addition to %s, print caller
//	%+v   extended format. Each Frame of the error's StackTrace will be printed in detail.
func (e *Error) Format(s fmt.State, verb rune) {
    if e != nil {
        mes := e.Error()
        switch verb {
        case 'v':
            mes = strings.Join([]string{mes, ", caller=[", e.Caller, "]"}, "")
            if s.Flag('+') {
                mes = strings.Join([]string{mes, ", trace=[", e.Trace, "]"}, "")
            }
            _, _ = fmt.Fprint(s, mes)
        case 's':
            _, _ = fmt.Fprint(s, mes)
        case 'q':
            _, _ = fmt.Fprint(s, mes)
        }
    }
}

// GetTypedMess Добавить типизированное сообщение
func GetTypedMess(key string, args ...interface{}) (code string, text string) {
    if v, ok := globalTypedErrorMessages[key]; ok {
        code = v.Code
        text = fmt.Sprintf(v.Text, args...)
    } else {
        v := globalTypedErrorMessages[ERR_MESSAGE_NOT_FOUND]
        text = fmt.Sprintf(v.Text, key, GetArgsString(args...))
    }

    return code, text
}

// Error print custom error
func (e *Error) Error() string {
    if e != nil {
        mes := strings.Join([]string{"errID=[", strconv.FormatUint(e.ID, 10), "], extID=[", strconv.FormatUint(e.ExternalId, 10), "], code=[", e.Code, "], message=[", e.Message, "]"}, "")
        if e.Details != nil && len(e.Details) > 0 {
            mes = fmt.Sprintf("%s, details=[%+v]", mes, e.Details)
        }
        if e.CauseMessage != "" {
            mes = strings.Join([]string{mes, ", cause_message=[", e.CauseMessage, "]"}, "")
        }
        return mes
    }
    return ""
}

// PrintfDebug print custom error
func (e *Error) PrintfDebug(depths ...int) *Error {
    if e != nil {
        depth := 1
        if len(depths) == 1 {
            depth = depth + depths[0]
        }
        _log.Log(_log.LEVEL_DEBUG, depth, e.Error())
        return e
    }
    return nil
}

// PrintfInfo print custom error
func (e *Error) PrintfInfo(depths ...int) *Error {
    if e != nil {
        depth := 1
        if len(depths) == 1 {
            depth = depth + depths[0]
        }
        _log.Log(_log.LEVEL_INFO, depth, e.Error())
        return e
    }
    return nil
}

// PrintfError print custom error
func (e *Error) PrintfError(depths ...int) *Error {
    if e != nil {
        depth := 1
        if len(depths) == 1 {
            depth = depth + depths[0]
        }
        _log.Log(_log.LEVEL_ERROR, depth, e.Error())
        return e
    }
    return nil
}

// GetTrace - напечатать trace
func GetTrace() string {
    return fmt.Sprintf("'%+v'", pkgerr.New(""))
}

// New - create new custom error
func New(code string, msg string, args ...interface{}) *Error {
    err := Error{
        ID:         getNextErrorID(),
        ExternalId: ERR_UNDEFINED_ID,
        Code:       code,
        Message:    msg,
        Caller:     _log.GetCaller(4),
        Args:       GetArgsString(args...),               // get formatted string with arguments
        Trace:      fmt.Sprintf("'%+v'", pkgerr.New("")), // create err and print it trace
    }

    return &err
}

// NewTyped - create new typed custom error
func NewTyped(key string, externalId uint64, args ...interface{}) *Error {
    code, msg := GetTypedMess(key, args...)
    err := New(code, msg, args...)
    err.ExternalId = externalId
    return err
}

// WithCause - create new custom error with cause
func WithCause(code string, msg string, causeErr error, args ...interface{}) *Error {
    err := New(code, msg, args...)
    err.CauseMessage = fmt.Sprintf("'%+v'", causeErr) // get formatted string from cause error
    err.CauseErr = causeErr
    return err
}

// WithCauseTyped - create new custom error with cause
func WithCauseTyped(key string, externalId uint64, causeErr error, args ...interface{}) *Error {
    code, msg := GetTypedMess(key, args...)
    err := WithCause(code, msg, causeErr, args...)
    err.ExternalId = externalId
    return err
}

// GetArgsString return formated string with arguments
func GetArgsString(args ...interface{}) (argsStr string) {
    for _, arg := range args {
        if arg != nil {
            argsStr = argsStr + fmt.Sprintf("'%v', ", arg)
        }
    }
    argsStr = strings.TrimRight(argsStr, ", ")
    return
}
