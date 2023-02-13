package httpservice

import (
	"net/http"

	"github.com/dgrijalva/jwt-go"
	auth "gopkg.in/korylprince/go-ad-auth.v2"

    _err "github.com/romapres2010/goapp/pkg/common/error"
    _jwt "github.com/romapres2010/goapp/pkg/common/jwt"
    _log "github.com/romapres2010/goapp/pkg/common/logger"
)

// checkAuthentication check HTTP Basic Authentication or MS AD Authentication
func (s *Service) checkAuthentication(reqID uint64, username string, password string) error {

	// В режиме "INTERNAL" сравним пользователя и пароль с тем что был передан при старте
	if s.cfg.AuthType == AUTH_TYPE_INTERNAL {
		if s.cfg.HTTPUser != username || s.cfg.HTTPPass != password {
			return _err.NewTyped(_err.ERR_HTTP_AUTH_BASIC_ERROR, reqID, username).PrintfError()
		}
		_log.Info("Success Internal Authentication: username", username)

	} else if s.cfg.AuthType == AUTH_TYPE_MSAD {
		config := &auth.Config{
			Server:   s.cfg.MSADServer,
			Port:     s.cfg.MSADPort,
			BaseDN:   s.cfg.MSADBaseDN,
			Security: auth.SecurityType(s.cfg.MSADSecurity),
		}

		status, err := auth.Authenticate(config, username, password)

		if err != nil {
			return _err.WithCauseTyped(_err.ERR_HTTP_AUTH_MSAD_INVALID_ERROR, reqID, err, s.cfg.MSADServer, s.cfg.MSADPort, s.cfg.MSADBaseDN, s.cfg.MSADSecurity, username).PrintfError()
		}

		if !status {
			return _err.NewTyped(_err.ERR_HTTP_AUTH_MSAD_ERROR, reqID, s.cfg.MSADServer, s.cfg.MSADPort, s.cfg.MSADBaseDN, s.cfg.MSADSecurity, username).PrintfError()
		}

		_log.Info("Success MS AD Authentication: username", username)
	} else {
		return _err.NewTyped(_err.ERR_INCORRECT_AUTH_TYPE, reqID, s.cfg.AuthType).PrintfError()
	}

	return nil
}

// SingingHandler handle authentication and creating JWT
func (s *Service) SingingHandler(w http.ResponseWriter, r *http.Request) {
	_log.Debug("START   ==================================================================================")

	// Получить уникальный номер HTTP запроса
	reqID := GetNextHTTPRequestID()

	// Считаем из заголовка HTTP Basic Authentication
	username, password, ok := r.BasicAuth()
	if !ok {
		myerr := _err.NewTyped(_err.ERR_HTTP_AUTH_BASIC_NOT_SET_ERROR, reqID).PrintfError()
		s.processError(myerr, w, r, http.StatusUnauthorized, reqID)
		return
	}
	_log.Debug("Get Authorization header: reqID, username", reqID, username)

	// Выполняем аутентификацию
	if myerr := s.checkAuthentication(reqID, username, password); myerr != nil {
		_log.ErrorAsInfo(myerr)
		s.processError(myerr, w, r, http.StatusUnauthorized, reqID)
		return
	}

	// Включен режим JSON web token (JWT)
	if s.cfg.UseJWT {

		// Create the JWT claims with username
		claims := &_jwt.Claims{
			Username:       username,
			StandardClaims: jwt.StandardClaims{},
		}

		// создадим новый токен и запишем его в Cookie
		_log.Debug("Create new JSON web token: reqID", reqID)
		cookie, myerr := _jwt.CreateJWTCookie(claims, s.cfg.JWTExpiresAt, s.cfg.JwtKey)
		if myerr != nil {
			_log.ErrorAsInfo(myerr)
			s.processError(myerr, w, r, http.StatusInternalServerError, reqID)
			return
		}

		// set the client cookie for "token" as the JWT
		http.SetCookie(w, cookie)

		_log.Debug("Set HTTP Cookie: reqID, cookie", reqID, cookie)
	} else {
		_log.Debug("JWT is off. Nothing to do: reqID", reqID)
	}

	_log.Debug("SUCCESS ==================================================================================")
}

// JWTRefreshHandler handle renew JWT
func (s *Service) JWTRefreshHandler(w http.ResponseWriter, r *http.Request) {
	_log.Debug("START   ==================================================================================")

	// Получить уникальный номер HTTP запроса
	reqID := GetNextHTTPRequestID()

	// Если включен режим JSON web token (JWT)
	if s.cfg.UseJWT {
		// проверим текущий JWT
		_log.Debug("JWT is on. Check JSON web token: reqID", reqID)

		// Считаем token из requests cookies
		cookie, err := r.Cookie("token")
		if err != nil {
			myerr := _err.WithCauseTyped(_err.ERR_HTTP_AUTH_JWT_NOT_SET_ERROR, reqID, err).PrintfError()
			s.processError(myerr, w, r, http.StatusUnauthorized, reqID) // расширенное логирование ошибки в контексте HTTP
			return
		}

		// Проверим JWT в token
		claims, myerr := _jwt.CheckJWT(reqID, cookie.Value, s.cfg.JwtKey)
		if myerr != nil {
			_log.ErrorAsInfo(myerr)
			s.processError(myerr, w, r, http.StatusUnauthorized, reqID) // расширенное логирование ошибки в контексте HTTP
			return
		}

		// создадим новый токен и запишем его в Cookie
		_log.Debug("JWT is valid. Create new JSON web token: reqID", reqID)
		if cookie, myerr = _jwt.CreateJWTCookie(claims, s.cfg.JWTExpiresAt, s.cfg.JwtKey); myerr != nil {
			_log.ErrorAsInfo(myerr)
			s.processError(myerr, w, r, http.StatusInternalServerError, reqID)
			return
		}

		// set the client cookie for "token" as the JWT
		http.SetCookie(w, cookie)

		_log.Debug("Set HTTP Cookie: reqID, cookie", reqID, cookie)

	} else {
		_log.Debug("JWT is off. Nothing to do: reqID", reqID)
	}

	_log.Debug("SUCCESS ==================================================================================")
}
