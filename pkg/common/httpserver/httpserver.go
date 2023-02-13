package httpserver

import (
	"context"
	"crypto/tls"
	"net"
	"net/http"
	"net/http/pprof"
	"time"

	"github.com/gorilla/mux"
	"github.com/prometheus/client_golang/prometheus/promhttp"

	//"crypto/go.cypherpunks.ru/gogost/v5/gost3410"

    _err "github.com/romapres2010/goapp/pkg/common/error"
    _http "github.com/romapres2010/goapp/pkg/common/httpservice"
    _log "github.com/romapres2010/goapp/pkg/common/logger"
    _metrics "github.com/romapres2010/goapp/pkg/common/metrics"
    _recover "github.com/romapres2010/goapp/pkg/common/recover"
)

// Server represent HTTP server
type Server struct {
	ctx    context.Context    // контекст при инициации сервиса
	cancel context.CancelFunc // функция закрытия контекста
	cfg    *Config            // конфигурация HTTP сервера
	errCh  chan<- error       // канал ошибок
	stopCh chan struct{}      // канал подтверждения об успешном закрытии HTTP сервера

	// вложенные сервисы
	listener    net.Listener   // листинер HTTP сервера
	router      *mux.Router    // роутер HTTP сервера
	httpServer  *http.Server   // собственно HTTP сервер
	httpService *_http.Service // сервис HTTP запросов
}

// Config represent HTTP server options
type Config struct {
	ListenSpec      string        `yaml:"listen_spec" json:"listen_spec"`           // HTTP listener address string
	ReadTimeout     time.Duration `yaml:"read_timeout" json:"read_timeout"`         // HTTP read timeout duration in sec - default 60 sec
	WriteTimeout    time.Duration `yaml:"write_timeout" json:"write_timeout"`       // HTTP write timeout duration in sec - default 60 sec
	IdleTimeout     time.Duration `yaml:"idle_timeout" json:"idle_timeout"`         // HTTP idle timeout duration in sec - default 60 sec
	ShutdownTimeout time.Duration `yaml:"shutdown_timeout" json:"shutdown_timeout"` // service shutdown timeout in sec - default 30 sec
	MaxHeaderBytes  int           `yaml:"max_header_bytes" json:"max_header_bytes"` // HTTP max header bytes - default 1 MB
	UseProfile      bool          `yaml:"use_go_profile" json:"use_profile"`        // use Go profiling
	UseTLS          bool          `yaml:"use_tls" json:"use_tls"`                   // use Transport Level Security
	TLSCertFile     string        `yaml:"tls_cert_file" json:"tls_cert_file"`       // TLS Certificate file name
	TLSKeyFile      string        `yaml:"tls_key_file" json:"tls_key_file"`         // TLS Private key file name
	TLSMinVersion   uint16        `yaml:"tls_min_version" json:"tls_min_version"`   // TLS min version VersionTLS13, VersionTLS12, VersionTLS11, VersionTLS10, VersionSSL30
	TLSMaxVersion   uint16        `yaml:"tls_max_version" json:"tls_max_version"`   // TLS max version VersionTLS13, VersionTLS12, VersionTLS11, VersionTLS10, VersionSSL30
}

// New create HTTP server
func New(ctx context.Context, errCh chan<- error, cfg *Config, httpService *_http.Service) (*Server, error) {
	var err error
	_log.Info("Creating new HTTP server")

	{ // входные проверки
		if cfg == nil {
			return nil, _err.NewTyped(_err.ERR_INCORRECT_CALL_ERROR, _err.ERR_UNDEFINED_ID, "if cfg == nil {}").PrintfError()
		}
		if cfg.ListenSpec == "" {
			return nil, _err.NewTyped(_err.ERR_INCORRECT_CALL_ERROR, _err.ERR_UNDEFINED_ID, "cfg.ArgsHTTPListenSpec == nil").PrintfError()
		}
		if httpService == nil {
			return nil, _err.NewTyped(_err.ERR_INCORRECT_CALL_ERROR, _err.ERR_UNDEFINED_ID, "httpService == nil").PrintfError()
		}
	} // входные проверки

	// Создаем новый сервер
	server := &Server{
		cfg:         cfg,
		errCh:       errCh,
		httpService: httpService,
		stopCh:      make(chan struct{}, 1), // канал подтверждения об успешном закрытии HTTP сервера
	}

	// создаем контекст с отменой
	if ctx == nil {
		server.ctx, server.cancel = context.WithCancel(context.Background())
	} else {
		server.ctx, server.cancel = context.WithCancel(ctx)
	}

	{ // Конфигурация HTTP сервера
		server.httpServer = &http.Server{
			ReadTimeout:  cfg.ReadTimeout,
			WriteTimeout: cfg.WriteTimeout,
			IdleTimeout:  cfg.IdleTimeout,
		}

		// Если задано ограничение на header
		if cfg.MaxHeaderBytes > 0 {
			server.httpServer.MaxHeaderBytes = cfg.MaxHeaderBytes
		}

		// настраиваем параметры TLS
		if server.cfg.UseTLS {
			//{ // ГОСТ сертификат
			//    serverCert, serverCertDer, serverPrv, _ := gostCertGen("server.gost.example.com", gost3410.CurveIdtc26gost341012256paramSetA(), x509.GOST256)
			//    //clientCert, clientCertDer, clientPrv, _ := gostCertGen("client.gost.example.com", gost3410.CurveIdtc26gost341012512paramSetC(), x509.GOST512)
			//    _ = serverCert
			//
			//    // подключение ГОСТ TLS
			//    tlsCfg := &tls.Config{
			//        MinVersion: tls.VersionTLS13,
			//        MaxVersion: tls.VersionTLS13,
			//        CurvePreferences: []tls.CurveID{
			//            tls.GOSTCurve256A,
			//            tls.GOSTCurve256B,
			//            tls.GOSTCurve256C,
			//            tls.GOSTCurve256D,
			//            tls.GOSTCurve512A,
			//            tls.GOSTCurve512B,
			//            tls.GOSTCurve512C,
			//        },
			//        CipherSuites: []uint16{
			//            tls.TLS_GOSTR341112_256_WITH_KUZNYECHIK_MGM_L,
			//            tls.TLS_GOSTR341112_256_WITH_MAGMA_MGM_L,
			//            tls.TLS_GOSTR341112_256_WITH_KUZNYECHIK_MGM_S,
			//            tls.TLS_GOSTR341112_256_WITH_MAGMA_MGM_S,
			//        },
			//        ClientAuth: tls.RequireAndVerifyClientCert,
			//        ClientCAs:  x509.NewCertPool(),
			//        Certificates: []tls.Certificate{{
			//            Certificate: [][]byte{serverCertDer},
			//            PrivateKey:  serverPrv,
			//        }},
			//    }
			//    server.httpServer.TLSConfig = tlsCfg
			//} // ГОСТ сертификат

			{
				// определим минимальную и максимальную версию TLS
				tlsCfg := &tls.Config{
					MinVersion: server.cfg.TLSMinVersion,
					MaxVersion: server.cfg.TLSMaxVersion,
					/*
					   	//отключение методов шифрования с ключами менее 256 бит
					       CurvePreferences:  []tls.CurveID{tls.CurveP521, tls.CurveP384, tls.CurveP256},
					       PreferServerCipherSuites: true,
					       CipherSuites: []uint16{
					           tls.TLS_ECDHE_RSA_WITH_AES_256_GCM_SHA384,
					           tls.TLS_ECDHE_RSA_WITH_AES_256_CBC_SHA,
					           tls.TLS_RSA_WITH_AES_256_GCM_SHA384,
					           tls.TLS_RSA_WITH_AES_256_CBC_SHA,
					       },
					*/
				}
				server.httpServer.TLSConfig = tlsCfg
				/*
					//Отключение HTTP/2, чтобы исключить поддержку ключа с 128 битами TLS_ECDHE_RSA_WITH_AES_128_GCM_SHA256
					server.httpServer.TLSNextProto = make(map[string]func(*http.Server, *tls.Conn, http.Handler), 0)
				*/
			}
		}
	} // Конфигурация HTTP сервера

	{ // Определяем листенер
		server.listener, err = net.Listen("tcp", cfg.ListenSpec)
		if err != nil {
			return nil, _err.WithCauseTyped(_err.ERR_HTTP_LISTENER_CREATE_ERROR, _err.ERR_UNDEFINED_ID, err, cfg.ListenSpec).PrintfError()
		}
		_log.Info("Created new TCP listener: network = 'tcp', address", cfg.ListenSpec)
	} // Определяем листенер

	{ // Настраиваем роутер
		// создаем новый роутер
		server.router = mux.NewRouter()

		// Устанавливаем роутер в качестве корневого обработчика
		http.Handle("/", server.router)

		// Expose the registered _metrics via HTTP.
		server.router.Handle("/metrics", promhttp.HandlerFor(
			_metrics.GlobalMetrics.Registry,
			promhttp.HandlerOpts{
				// Opt into OpenMetrics to support exemplars.
				EnableOpenMetrics: true,
			},
		))

		// Зарегистрируем HTTP обработчиков
		if server.httpService.Handlers != nil {
			for _, h := range server.httpService.Handlers {
				server.router.HandleFunc(h.Path, h.HandlerFunc).Methods(h.Method)
				_log.Info("Handler is registered: Path, Method", h.Path, h.Method)
			}
		}

		// Регистрация pprof-обработчиков
		if server.cfg.UseProfile {
			_log.Info("'/debug/pprof' is registered")

			pprofrouter := server.router.PathPrefix("/debug/pprof").Subrouter()
			pprofrouter.HandleFunc("/", pprof.Index)
			pprofrouter.HandleFunc("/cmdline", pprof.Cmdline)
			pprofrouter.HandleFunc("/symbol", pprof.Symbol)
			pprofrouter.HandleFunc("/trace", pprof.Trace)

			profile := pprofrouter.PathPrefix("/profile").Subrouter()
			profile.HandleFunc("", pprof.Profile)
			profile.Handle("/goroutine", pprof.Handler("goroutine"))
			profile.Handle("/threadcreate", pprof.Handler("threadcreate"))
			profile.Handle("/heap", pprof.Handler("heap"))
			profile.Handle("/block", pprof.Handler("block"))
			profile.Handle("/mutex", pprof.Handler("mutex"))
		}
	} // Настраиваем роутер

	_log.Info("HTTP server is created")
	return server, nil
}

// Run HTTP server - wait for error or exit
func (s *Server) Run() (myerr error) {
	// Функция восстановления после паники
	defer func() {
		r := recover()
		if r != nil {
			myerr = _recover.GetRecoverError(r, _err.ERR_UNDEFINED_ID)
		}
		//if myerr != nil {
		//    s.errCh <- myerr // передаем ошибку явно в канал для уведомления daemon или через возврат функции Run
		//}
	}()

	// Запускаем HTTP сервер
	if s.cfg.UseTLS {
		_log.Info("Starting HTTPS server: TLSSertFile, TLSKeyFile", s.cfg.TLSCertFile, s.cfg.TLSKeyFile)
		return s.httpServer.ServeTLS(s.listener, s.cfg.TLSCertFile, s.cfg.TLSKeyFile)
	} else {
		_log.Info("Starting HTTP server")
		return s.httpServer.Serve(s.listener)
	}
}

// Shutdown HTTP server
func (s *Server) Shutdown() (myerr error) {
	_log.Info("Waiting for shutdown HTTP Server: sec", s.cfg.ShutdownTimeout)

	// подтверждение о закрытии HTTP сервера
	defer func() { s.stopCh <- struct{}{} }()

	// закроем контекст HTTP сервера
	defer s.cancel()

	// создаем новый контекст с отменой и отсрочкой ShutdownTimeout
	cancelCtx, cancel := context.WithTimeout(s.ctx, s.cfg.ShutdownTimeout)
	defer cancel()

	// ожидаем закрытия активных подключений в течении ShutdownTimeout
	if err := s.httpServer.Shutdown(cancelCtx); err != nil {
		myerr = _err.WithCauseTyped(_err.ERR_HTTP_SERVER_SHUTDOWN_ERROR, _err.ERR_UNDEFINED_ID, err, s.cfg.ShutdownTimeout).PrintfError()

		return
	}

	_log.Info("HTTP Server shutdown successfully")
	return
}

//func gostCertGen(
//    commonName string,
//    curve *gost3410.Curve,
//    sigAlgo x509.SignatureAlgorithm) (*x509.Certificate, []byte, crypto.PrivateKey, []byte) {
//    prvRaw := make([]byte, curve.PointSize())
//    if _, err := io.ReadFull(rand.Reader, prvRaw); err != nil {
//        _log.Error("can not read PrivateKey random", err)
//    }
//    prv, err := gost3410.NewPrivateKey(curve, prvRaw)
//    if err != nil {
//        _log.Error("can not generate PrivateKey", err)
//    }
//    pub, err := prv.PublicKey()
//    if err != nil {
//        _log.Error("can not generate PublicKey", err)
//    }
//    template := x509.Certificate{
//        SerialNumber:       big.NewInt(12345),
//        Subject:            pkix.Name{CommonName: commonName},
//        DNSNames:           []string{commonName},
//        NotBefore:          time.Date(2020, 1, 1, 0, 0, 0, 0, time.UTC),
//        NotAfter:           time.Date(2030, 1, 1, 0, 0, 0, 0, time.UTC),
//        SignatureAlgorithm: sigAlgo,
//    }
//    certDer, err := x509.CreateCertificate(
//        rand.Reader,
//        &template, &template,
//        pub, &gost3410.PrivateKeyReverseDigest{Prv: prv},
//    )
//    if err != nil {
//        _log.Error("can not generate certificate", err)
//    }
//    cert, err := x509.ParseCertificate(certDer)
//    if err != nil {
//        _log.Error("can not parse generated certificate", err)
//    }
//    return cert, certDer, &gost3410.PrivateKeyReverseDigestAndSignature{Prv: prv}, prvRaw
//}
