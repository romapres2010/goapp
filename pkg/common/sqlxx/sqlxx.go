package sqlxx

import (
	"database/sql"
	"fmt"
	"reflect"
	"sync/atomic"
	"time"

	_ "github.com/jackc/pgx/stdlib"
	"github.com/jmoiron/sqlx"
	_ "github.com/lib/pq"

    _err "github.com/romapres2010/goapp/pkg/common/error"
    _log "github.com/romapres2010/goapp/pkg/common/logger"
    _metrics "github.com/romapres2010/goapp/pkg/common/metrics"
    _recover "github.com/romapres2010/goapp/pkg/common/recover"
)

// Config конфигурационные настройки БД
type Config struct {
	ConnectString   string        `yaml:"-" json:"-"`                                 // строка подключения к БД
	Host            string        `yaml:"host" json:"host"`                           // host БД
	Port            string        `yaml:"port" json:"port"`                           // порт БД
	Dbname          string        `yaml:"dbname" json:"dbname"`                       // имя БД
	SslMode         string        `yaml:"ssl_mode" json:"ssl_mode"`                   // режим SSL
	User            string        `yaml:"user" json:"user"`                           // пользователь БД
	Pass            string        `yaml:"pass" json:"pass"`                           // пароль пользователя БД
	ConnMaxLifetime time.Duration `yaml:"conn_max_lifetime" json:"conn_max_lifetime"` // время жизни подключения в миллисекундах
	MaxOpenConns    int           `yaml:"max_open_conns" json:"max_open_conns"`       // максимальное количество открытых подключений
	MaxIdleConns    int           `yaml:"max_idle_conns" json:"max_idle_conns"`       // максимальное количество простаивающих подключений
	DriverName      string        `yaml:"driver_name" json:"driver_name"`             // имя драйвера "postgres" | "pgx" | "godror"
}

// DB is a wrapper around sqlx.DB
type DB struct {
	*sqlx.DB

	cfg     *Config
	sqlStms SQLStms // SQL команды
}

// Tx is an sqlx wrapper around sqlx.Tx
type Tx struct {
	*sqlx.Tx
}

// SQLStm represent SQL text and sqlStm
type SQLStm struct {
	Text        string     // текст SQL команды
	Stmt        *sqlx.Stmt // подготовленная SQL команда
	NeedPrepare bool       // признак, нужно ли предварительно готовить SQL команду
}

// SQLStms represent SQLStm map
type SQLStms map[string]*SQLStm

// уникальный номер SQL
var sqlIDGlobal uint64

// GetNextSQLID - запросить номер следующей SQL
func GetNextSQLID() uint64 {
	return atomic.AddUint64(&sqlIDGlobal, 1)
}

// New - create new connect to DB
func New(cfg *Config, sqlStms SQLStms) (db *DB, myerr error) {
	{ // входные проверки
		if cfg == nil {
			return nil, _err.NewTyped(_err.ERR_INCORRECT_CALL_ERROR, _err.ERR_UNDEFINED_ID, "if cfg == nil {}").PrintfError()
		}
	} // входные проверки

	// Сформировать строку подключения
	cfg.ConnectString = fmt.Sprintf("host=%s port=%s dbname=%s sslmode=%s user=%s password=%s ", cfg.Host, cfg.Port, cfg.Dbname, cfg.SslMode, cfg.User, cfg.Pass)

	// Создаем новый сервис
	db = &DB{
		cfg: cfg,
	}

	if sqlStms != nil {
		db.sqlStms = sqlStms
	} else {
		db.sqlStms = make(map[string]*SQLStm)
	}

	// открываем соединение с БД
	_log.Info("Testing connect to DB server: host, port, dbname, sslmode, user", cfg.Host, cfg.Port, cfg.Dbname, cfg.SslMode, cfg.User)

	sqlxDb, err := sqlx.Connect(cfg.DriverName, cfg.ConnectString)
	if err != nil {
		return nil, _err.WithCauseTyped(_err.ERR_DB_CONNECTION_ERROR, _err.ERR_UNDEFINED_ID, err, cfg.Host, cfg.Port, cfg.Dbname, cfg.SslMode, cfg.User).PrintfError()
	}
	db.DB = sqlxDb

	{ // Устанавливаем параметры пула подключений
		db.DB.SetMaxOpenConns(cfg.MaxOpenConns)
		db.DB.SetMaxIdleConns(cfg.MaxIdleConns)
		db.DB.SetConnMaxLifetime(cfg.ConnMaxLifetime)
	} // Устанавливаем параметры пула подключений

	// Подготовим SQL команды
	if err = db.PreparexAll(sqlStms); err != nil {
		return nil, err
	}

	_log.Info("Success connect to DB server")
	return db, nil
}

// Close - close DB
func (db *DB) Close() (myerr error) {
	if db == nil || db.DB == nil {
		return _err.NewTyped(_err.ERR_INCORRECT_CALL_ERROR, _err.ERR_UNDEFINED_ID, "if db == nil || db.DB == nil {}").PrintfError()
	}

	myerr = db.DB.Close()
	if myerr != nil {
		return _err.WithCauseTyped(_err.ERR_DB_CLOSE_ERROR, _err.ERR_UNDEFINED_ID, myerr, db.cfg.Host, db.cfg.Port, db.cfg.Dbname, db.cfg.SslMode, db.cfg.User).PrintfError()
	}
	_log.Info("Success close DB server")
	return nil
}

// PreparexAll - prepare SQL statements
func (db *DB) PreparexAll(sqlStms SQLStms) (myerr error) {
	var err error

	if db == nil || db.DB == nil {
		return _err.NewTyped(_err.ERR_INCORRECT_CALL_ERROR, _err.ERR_UNDEFINED_ID, "if db == nil || db.DB == nil {}").PrintfError()
	}

	// Подготовим SQL команды
	for _, h := range sqlStms {
		if h.NeedPrepare {
			if h.Stmt, err = db.DB.Preparex(h.Text); err != nil {
				return _err.WithCauseTyped(_err.ERR_DB_SQL_STM_PREPARE_ERROR, _err.ERR_UNDEFINED_ID, err, h.Text).PrintfError()
			}
			_log.Info("SQL statement was prepared: SQL", h.Text)
		}
	}
	return nil
}

// PreparexAddSql - prepare SQL statements and add it to map
func (db *DB) PreparexAddSql(sql string, needPrepare bool) (myerr error) {
	var err error

	if db == nil || db.DB == nil {
		return _err.NewTyped(_err.ERR_INCORRECT_CALL_ERROR, _err.ERR_UNDEFINED_ID, "if db == nil || db.DB == nil {}").PrintfError()
	}

	if sql != "" {
		var sqlStm *SQLStm
		var ok bool

		// Ищем полное совпадение команды, если не находим, создаем новую
		if sqlStm, ok = db.sqlStms[sql]; ok {
			// Уже подготовлена
			if sqlStm.Stmt != nil {
				_log.Debug("SQL statement was ALREADY prepared: SQL", sqlStm.Text)
				return nil
			}
		} else {
			sqlStm = &SQLStm{
				Text:        sql,
				NeedPrepare: needPrepare,
			}
		}

		// Подготовим SQL команды
		if sqlStm.NeedPrepare {
			if sqlStm.Stmt, err = db.DB.Preparex(sqlStm.Text); err != nil {
				return _err.WithCauseTyped(_err.ERR_DB_SQL_STM_PREPARE_ERROR, _err.ERR_UNDEFINED_ID, err, sqlStm.Text).PrintfError()
			}
			_log.Debug("SQL statement was prepared: SQL", sqlStm.Text)
		}
		db.sqlStms[sql] = sqlStm
	}
	return nil
}

// Beginx - begin a new transaction
func (db *DB) Beginx(externalId uint64) (tx *Tx, myerr error) {
	// функция восстановления после паники
	defer func() {
		r := recover()
		if r != nil {
			myerr = _recover.GetRecoverError(r, externalId)
		}
	}()

	if db == nil || db.DB == nil {
		return nil, _err.NewTyped(_err.ERR_INCORRECT_CALL_ERROR, externalId, "if db == nil || db.DB == nil {}").PrintfError()
	}

	sqlxTx, err := db.DB.Beginx()
	if err != nil {
		return nil, _err.WithCauseTyped(_err.ERR_DB_TX_CREATE_ERROR, externalId, err).PrintfInfo()
	}
	return &Tx{sqlxTx}, nil
}

// Rollback - rollback the transaction
func (db *DB) Rollback(externalId uint64, tx *Tx) (myerr error) {
	// функция восстановления после паники
	defer func() {
		r := recover()
		if r != nil {
			myerr = _recover.GetRecoverError(r, externalId)
		}
	}()

	if db == nil || db.DB == nil {
		return _err.NewTyped(_err.ERR_INCORRECT_CALL_ERROR, externalId, "if db == nil || db.DB == nil {}").PrintfError()
	}

	// Проверяем определен ли контекст транзакции
	if tx == nil {
		return _err.NewTyped(_err.ERR_DB_TX_NOT_DEFINED_ERROR, externalId).PrintfError()
	}

	if err := tx.Rollback(); err != nil {
		return _err.WithCauseTyped(_err.ERR_DB_TX_ROLLBAK_ERROR, externalId, err).PrintfError()
	}
	_log.Debug("Transaction was rollback", externalId)
	return nil
}

// Commit - commit the transaction
func (db *DB) Commit(externalId uint64, tx *Tx) (myerr error) {
	// функция восстановления после паники
	defer func() {
		r := recover()
		if r != nil {
			myerr = _recover.GetRecoverError(r, externalId)
		}
	}()

	if db == nil || db.DB == nil {
		return _err.NewTyped(_err.ERR_INCORRECT_CALL_ERROR, externalId, "if db == nil || db.DB == nil {}").PrintfError()
	}

	// Проверяем определен ли контекст транзакции
	if tx == nil {
		return _err.NewTyped(_err.ERR_DB_TX_NOT_DEFINED_ERROR, externalId).PrintfError()
	}

	if err := tx.Commit(); err != nil {
		return _err.WithCauseTyped(_err.ERR_DB_TX_COMMIT_ERROR, externalId, err).PrintfError()
	}
	_log.Debug("Transaction committed", externalId)
	return nil
}

// Select - represent common task in process SQL Select statement
func (db *DB) Select(externalId uint64, tx *Tx, sqlT string, dest interface{}, args ...interface{}) (myerr error) {
	sqlID := GetNextSQLID()

	if db == nil || db.DB == nil {
		return _err.NewTyped(_err.ERR_INCORRECT_CALL_ERROR, externalId, "if db == nil || db.DB == nil {}").PrintfError()
	}

	sqlStm, ok := db.sqlStms[sqlT]
	if !ok {
		return _err.NewTyped(_err.ERR_DB_SQL_STM_NOT_DEFINED_ERROR, externalId, sqlT).PrintfError()
	}

	// функция восстановления после паники
	defer func() {
		r := recover()
		if r != nil {
			arg := fmt.Sprintf("sqlID=[%v], sql=[%s], args=[%s]", []interface{}{sqlID, sqlStm.Text, args}...)
			myerr = _recover.GetRecoverError(r, externalId, arg)
		}
	}()

	if dest != nil && !reflect.ValueOf(dest).IsNil() {

		_log.Debug("SELECT - reqID, sqlID, SQL, args", []interface{}{externalId, sqlID, sqlStm.Text, args}...)

		stm := sqlStm.Stmt
		// Помещаем запрос в рамки транзакции
		if tx != nil {
			stm = tx.Stmtx(sqlStm.Stmt)
		}

		//Выполняем запрос
		var tic = time.Now()
		err := stm.Select(dest, args...)
		_metrics.IncDBCountVec("Select: " + sqlT)
		_metrics.AddDBDurationVec("Select: "+sqlT, time.Now().Sub(tic))
		if err != nil {
			return _err.WithCauseTyped(_err.ERR_DB_SELECT_ERROR, externalId, err, sqlID, sqlStm.Text, fmt.Sprintf("%s", args)).PrintfInfo()
		}
		return nil
	}
	return _err.NewTyped(_err.ERR_INCORRECT_CALL_ERROR, externalId, "dest != nil && !reflect.ValueOf(dest).IsNil()").PrintfError()
}

// Get - represent common task in process SQL Select statement with only one rows
func (db *DB) Get(externalId uint64, tx *Tx, sqlT string, dest interface{}, args ...interface{}) (exists bool, myerr error) {
	sqlID := GetNextSQLID()

	if db == nil || db.DB == nil {
		return false, _err.NewTyped(_err.ERR_INCORRECT_CALL_ERROR, externalId, "if db == nil || db.DB == nil {}").PrintfError()
	}

	sqlStm, ok := db.sqlStms[sqlT]
	if !ok {
		return false, _err.NewTyped(_err.ERR_DB_SQL_STM_NOT_DEFINED_ERROR, externalId, sqlT).PrintfError()
	}

	// функция восстановления после паники
	defer func() {
		r := recover()
		if r != nil {
			arg := fmt.Sprintf("sqlID=[%v], sql=[%s], args=[%s]", []interface{}{sqlID, sqlStm.Text, args}...)
			myerr = _recover.GetRecoverError(r, externalId, arg)
		}
	}()

	if dest != nil && !reflect.ValueOf(dest).IsNil() {
		_log.Debug("GET - reqID, sqlID, SQL, args", []interface{}{externalId, sqlID, sqlStm.Text, args}...)

		stm := sqlStm.Stmt
		// Помещаем запрос в рамки транзакции
		if tx != nil {
			stm = tx.Stmtx(sqlStm.Stmt)
		}

		//Выполняем запрос
		var tic = time.Now()
		err := stm.Get(dest, args...)
		_metrics.IncDBCountVec("Get: " + sqlT)
		_metrics.AddDBDurationVec("Get: "+sqlT, time.Now().Sub(tic))
		if err != nil {
			// NO_DATA_FOUND - ошибкой не считаем
			if err == sql.ErrNoRows {
				return false, nil
			}
			return false, _err.WithCauseTyped(_err.ERR_DB_GET_ERROR, externalId, err, sqlID, sqlStm.Text, fmt.Sprintf("%s", args)).PrintfInfo()
		}
		return true, nil
	}
	return false, _err.NewTyped(_err.ERR_INCORRECT_CALL_ERROR, externalId, "dest != nil && !reflect.ValueOf(dest).IsNil()").PrintfError()
}

// Exec - represent common task in process DML statement
func (db *DB) Exec(externalId uint64, tx *Tx, sqlT string, args interface{}) (rowsAffected int64, lastInsertId int64, myerr error) {
	sqlID := GetNextSQLID()

	if db == nil || db.DB == nil {
		return 0, 0, _err.NewTyped(_err.ERR_INCORRECT_CALL_ERROR, externalId, "if db == nil || db.DB == nil {}").PrintfError()
	}

	sqlStm, ok := db.sqlStms[sqlT]
	if !ok {
		return 0, 0, _err.NewTyped(_err.ERR_DB_SQL_STM_NOT_DEFINED_ERROR, externalId, sqlT).PrintfError()
	}

	// функция восстановления после паники
	defer func() {
		r := recover()
		if r != nil {
			arg := fmt.Sprintf("sqlID=[%v], sql=[%s], args=[%s]", []interface{}{sqlID, sqlStm.Text, args}...)
			myerr = _recover.GetRecoverError(r, externalId, arg)
		}
	}()

	if args != nil && !reflect.ValueOf(args).IsNil() {
		_log.Debug("EXEC - reqID, sqlID, SQL, args", []interface{}{externalId, sqlID, sqlStm.Text, args}...)

		// Проверяем определен ли контекст транзакции
		if tx == nil {
			return 0, 0, _err.NewTyped(_err.ERR_DB_TX_NOT_DEFINED_ERROR, externalId).PrintfError()
		}

		// Выполняем DML
		var tic = time.Now()
		res, err := tx.NamedExec(sqlStm.Text, args)
		_metrics.IncDBCountVec("NamedExec: " + sqlT)
		_metrics.AddDBDurationVec("NamedExec: "+sqlT, time.Now().Sub(tic))
		if err != nil {
			return 0, 0, _err.WithCauseTyped(_err.ERR_DB_EXEC_ERROR, externalId, err, sqlID, sqlStm.Text, fmt.Sprintf("%s", args)).PrintfInfo()
		}

		// Последняя обработанная id
		lastInsertId, _ = res.LastInsertId()
		//if err != nil {
		//	return 0, 0,err
		//}

		// Количество обработанных строк
		rowsAffected, err = res.RowsAffected()
		if err != nil {
			return 0, 0, _err.WithCauseTyped(_err.ERR_DB_EXEC_ROW_COUNT_ERROR, externalId, err, sqlID, sqlStm.Text, fmt.Sprintf("%s", args)).PrintfInfo()
		}
		return rowsAffected, lastInsertId, nil
	}
	return 0, 0, _err.NewTyped(_err.ERR_INCORRECT_CALL_ERROR, externalId, "if args != nil && !reflect.ValueOf(args).IsNil() {}").PrintfError()
}

// QueryRowxStructScan - represent common task in process SQL Select statement with only one rows
func (db *DB) QueryRowxStructScan(externalId uint64, tx *Tx, sqlT string, dest interface{}, args ...interface{}) (myerr error) {
	sqlID := GetNextSQLID()

	if db == nil || db.DB == nil {
		return _err.NewTyped(_err.ERR_INCORRECT_CALL_ERROR, externalId, "if db == nil || db.DB == nil {}").PrintfError()
	}

	sqlStm, ok := db.sqlStms[sqlT]
	if !ok {
		return _err.NewTyped(_err.ERR_DB_SQL_STM_NOT_DEFINED_ERROR, externalId, sqlT).PrintfError()
	}

	// функция восстановления после паники
	defer func() {
		r := recover()
		if r != nil {
			arg := fmt.Sprintf("sqlID=[%v], sql=[%s], args=[%s]", []interface{}{sqlID, sqlStm.Text, args}...)
			myerr = _recover.GetRecoverError(r, externalId, arg)
		}
	}()

	if dest != nil && !reflect.ValueOf(dest).IsNil() {
		_log.Debug("QueryRowxStructScan - reqID, sqlID, SQL, args", []interface{}{externalId, sqlID, sqlStm.Text, args}...)

		// Проверяем определен ли контекст транзакции
		if tx == nil {
			return _err.NewTyped(_err.ERR_DB_TX_NOT_DEFINED_ERROR, externalId).PrintfError()
		}

		// Выполняем DML
		var tic = time.Now()
		err := tx.QueryRowx(sqlStm.Text, args...).StructScan(dest)
		_metrics.IncDBCountVec("QueryRowxStructScan.StructScan: " + sqlT)
		_metrics.AddDBDurationVec("QueryRowxStructScan.StructScan: "+sqlT, time.Now().Sub(tic))
		if err != nil {
			return _err.WithCauseTyped(_err.ERR_DB_QUERYROWX_ERROR, externalId, err, sqlID, sqlStm.Text, fmt.Sprintf("%s", args)).PrintfInfo()
		}

		return nil
	}
	return _err.NewTyped(_err.ERR_INCORRECT_CALL_ERROR, externalId, "dest != nil && !reflect.ValueOf(dest).IsNil()").PrintfError()
}
