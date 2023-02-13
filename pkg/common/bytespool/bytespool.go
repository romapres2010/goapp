package bytespool

import (
	"sync"
	"sync/atomic"

    _err "github.com/romapres2010/goapp/pkg/common/error"
    _log "github.com/romapres2010/goapp/pkg/common/logger"
)

// Pool represent pooling of []byte
type Pool struct {
	cfg  *Config // конфигурационные параметры
	pool sync.Pool
}

// Config - represent BytesPool Service configurations
type Config struct {
	PooledSize int `yaml:"pooled_size" json:"pooled_size"`
}

// Represent a pool statistics for benchmarking
var (
	countGet uint64 // количество запросов кэша
	countPut uint64 // количество возвратов в кэша
	countNew uint64 // количество создания нового объекта
)

// New create new BytesPool
func New(cfg *Config) *Pool {
	if cfg != nil {
		p := &Pool{
			cfg: cfg,
			pool: sync.Pool{
				New: func() interface{} {
					atomic.AddUint64(&countNew, 1)
					return make([]byte, cfg.PooledSize)
				},
			},
		}
		return p
	} else {
		_ = _err.NewTyped(_err.ERR_INCORRECT_CALL_ERROR, _err.ERR_UNDEFINED_ID, "cfg != nil").PrintfError()
		return nil
	}
}

// GetBuf allocates a new []byte
func (p *Pool) GetBuf() []byte {
	if p != nil {
		atomic.AddUint64(&countGet, 1)
		return p.pool.Get().([]byte)
	} else {
		_ = _err.NewTyped(_err.ERR_INCORRECT_CALL_ERROR, _err.ERR_UNDEFINED_ID, "p != nil").PrintfError()
		return nil
	}
}

// PutBuf return byte buf to cache
func (p *Pool) PutBuf(buf []byte) {
	if p != nil {
		size := cap(buf)
		if size < p.cfg.PooledSize { // не выгодно хранить маленькие буферы
			return
		}
		atomic.AddUint64(&countPut, 1)
		p.pool.Put(buf[:0])
	} else {
		_ = _err.NewTyped(_err.ERR_INCORRECT_CALL_ERROR, _err.ERR_UNDEFINED_ID, "p != nil").PrintfError()
	}
}

// PrintBytesPoolStats print statistics about bytes pool
func (p *Pool) PrintBytesPoolStats() {
	if p != nil {
		_log.Info("Usage bytes pool: countGet, countPut, countNew", countGet, countPut, countNew)
	} else {
		_ = _err.NewTyped(_err.ERR_INCORRECT_CALL_ERROR, _err.ERR_UNDEFINED_ID, "p != nil").PrintfError()
	}
}
