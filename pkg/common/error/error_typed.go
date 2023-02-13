package errors

const ERR_MESSAGE_NOT_FOUND = "MESSAGE_NOT_FOUND"
const ERR_INCORRECT_CALL_ERROR = "INCORRECT_CALL_ERROR"
const ERR_INCORRECT_TYPE_ERROR = "INCORRECT_TYPE_ERROR"
const ERR_INCORRECT_LOG_LEVEL = "INCORRECT_LOG_LEVEL"

const ERR_JSON_MARSHAL_ERROR = "JSON_MARSHAL_ERROR"
const ERR_JSON_UNMARSHAL_ERROR = "JSON_UNMARSHAL_ERROR"
const ERR_XML_MARSHAL_ERROR = "XML_MARSHAL_ERROR"
const ERR_XML_UNMARSHAL_ERROR = "XML_UNMARSHAL_ERROR"
const ERR_YAML_MARSHAL_ERROR = "YAML_MARSHAL_ERROR"
const ERR_YAML_UNMARSHAL_ERROR = "YAML_UNMARSHAL_ERROR"

const ERR_HTTP_LISTENER_CREATE_ERROR = "HTTP_LISTENER_CREATE_ERROR"
const ERR_HTTP_SERVER_SHUTDOWN_ERROR = "HTTP_SERVER_SHUTDOWN_ERROR"
const ERR_HTTP_DUMP_REQUEST_ERROR = "HTTP_DUMP_REQUEST_ERROR"
const ERR_HTTP_BODY_READ_ERROR = "HTTP_BODY_READ_ERROR"
const ERR_HTTP_REQUEST_WRITE_ERROR = "HTTP_REQUEST_WRITE_ERROR"
const ERR_HTTP_METHOD_NOT_ALLOWED_ERROR = "HTTP_METHOD_NOT_ALLOWED_ERROR"
const ERR_HTTP_HANDLER_METHOD_NOT_FOUND = "HTTP_HANDLER_METHOD_NOT_FOUND"

const ERR_HTTP_AUTH_BASIC_NOT_SET_ERROR = "HTTP_AUTH_BASIC_NOT_SET_ERROR"
const ERR_HTTP_AUTH_BASIC_ERROR = "HTTP_AUTH_BASIC_ERROR"
const ERR_HTTP_AUTH_JWT_NOT_SET_ERROR = "HTTP_AUTH_JWT_NOT_SET_ERROR"
const ERR_HTTP_AUTH_JWT_INVALID_ERROR = "HTTP_AUTH_JWT_INVALID_ERROR"
const ERR_HTTP_AUTH_JWT_EXPIRED_ERROR = "HTTP_AUTH_JWT_EXPIRED_ERROR"
const ERR_HTTP_AUTH_MSAD_INVALID_ERROR = "HTTP_AUTH_MSAD_INVALID_ERROR"
const ERR_HTTP_AUTH_MSAD_ERROR = "HTTP_AUTH_MSAD_ERROR"
const ERR_HTTP_HEADER_INCORRECT_BOOL = "HTTP_HEADER_INCORRECT_BOOL"

const ERR_HTTP_CALL_EMPTY_URL = "HTTP_CALL_EMPTY_URL"
const ERR_HTTP_CALL_CREATE_CONTEXT_ERROR = "HTTP_CALL_CREATE_CONTEXT_ERROR"
const ERR_HTTP_CALL_TIMEOUT_ERROR = "HTTP_CALL_TIMEOUT_ERROR"
const ERR_HTTP_CALL_OTHER_ERROR = "HTTP_CALL_OTHER_ERROR"
const ERR_HTTP_CALL_OTHER_NETURL_ERROR = "HTTP_CALL_OTHER_NETURL_ERROR"
const ERR_HTTP_CALL_READ_BODY_ERROR = "HTTP_CALL_READ_BODY_ERROR"
const ERR_HTTP_CALL_URL_NOT_FOUND_ERROR = "HTTP_CALL_URL_NOT_FOUND_ERROR"
const ERR_HTTP_CALL_METHOD_NOT_ALLOWED_ERROR = "HTTP_CALL_METHOD_NOT_ALLOWED_ERROR"

const ERR_DB_CONNECTION_ERROR = "DB_CONNECTION_ERROR"
const ERR_DB_CLOSE_ERROR = "DB_CLOSE_ERROR"
const ERR_DB_TX_NOT_DEFINED_ERROR = "DB_TX_NOT_DEFINED_ERROR"
const ERR_DB_SQL_STM_NOT_DEFINED_ERROR = "DB_SQL_STM_NOT_DEFINED_ERROR"
const ERR_DB_SQL_STM_PREPARE_ERROR = "DB_SQL_STM_PREPARE_ERROR"
const ERR_DB_TX_COMMIT_ERROR = "DB_TX_COMMIT_ERROR"
const ERR_DB_TX_ROLLBAK_ERROR = "DB_TX_ROLLBAK_ERROR"
const ERR_DB_TX_CREATE_ERROR = "DB_TX_CREATE_ERROR"
const ERR_DB_SELECT_ERROR = "DB_SELECT_ERROR"
const ERR_DB_GET_ERROR = "DB_GET_ERROR"
const ERR_DB_EXEC_ERROR = "DB_EXEC_ERROR"
const ERR_DB_QUERYROWX_ERROR = "DB_QUERYROWX_ERROR"
const ERR_DB_EXEC_ROW_COUNT_ERROR = "DB_EXEC_ROW_COUNT_ERROR"
const ERR_DB_TOO_MANY_ROWS_ERROR = "DB_TOO_MANY_ROWS_ERROR"

const ERR_PANIC_RECOVER_ERROR = "PANIC_RECOVER_ERROR"

const ERR_WORKER_POOL_TIMEOUT_ERROR = "WORKER_POOL_TIMEOUT_ERROR"
const ERR_WORKER_POOL_WORKER_ERROR = "WORKER_POOL_WORKER_ERROR"
const ERR_WORKER_POOL_TASK_COUNT_ERROR = "WORKER_POOL_TASK_COUNT_ERROR"
const ERR_WORKER_POOL_IS_SHUTTING_DOWN = "WORKER_POOL_IS_SHUTTING_DOWN"
const ERR_WORKER_POOL_TASK_CHANNEL_CLOSED = "WORKER_POOL_TASK_CHANNEL_CLOSED"

const ERR_TIMEOUT_ERROR = "TIMEOUT_ERROR"

const ERR_INCORRECT_ARG_NUM_ERROR = "INCORRECT_ARG_NUM_ERROR"

const ERR_COMMON_ERROR = "COMMON_ERROR"
const ERR_ERROR = "ERROR"

const ERR_REFLECT_INCORRECT_TYPE = "REFLECT_INCORRECT_TYPE"

const ERR_CONTEXT_CLOSED_ERROR = "CONTEXT_CLOSED_ERROR"

const ERR_VALIDATOR_COMMON_ERROR = "VALIDATOR_COMMON_ERROR"
const ERR_VALIDATOR_STRUCT_ERROR = "VALIDATOR_STRUCT_ERROR"
const ERR_VALIDATOR_MDG_ERROR = "VALIDATOR_MDG_ERROR"

const ERR_FILE_OPEN_ERROR = "FILE_OPEN_ERROR"
const ERR_FILE_SYNC_ERROR = "FILE_SYNC_ERROR"

const ERR_XLSX_COMMON_ERROR = "XLSX_COMMON_ERROR"

const ERR_TIME_FORMAT_ERROR = "TIME_FORMAT_ERROR"
const ERR_INT_PARSE_ERROR = "INT_PARSE_ERROR"

const ERR_INCORRECT_ENTITY_SOURCE = "INCORRECT_ENTITY_SOURCE"
const ERR_INCORRECT_AUTH_TYPE = "INCORRECT_AUTH_TYPE"

const ERR_EMPTY_JWT_KEY = "EMPTY_JWT_KEY"
const ERR_EMPTY_HTTP_USER = "EMPTY_HTTP_USER"
const ERR_EMPTY_CONFIG_FILE = "EMPTY_CONFIG_FILE"
const ERR_CONFIG_FILE_NOT_EXISTS = "CONFIG_FILE_NOT_EXISTS"
const ERR_CONFIG_FILE_LOAD_ERROR = "CONFIG_FILE_LOAD_ERROR"

// globalTypedErrorMessages - типизированные сообщения
var globalTypedErrorMessages = map[string]ErrorMessage{
	ERR_MESSAGE_NOT_FOUND:                  {ERR_MESSAGE_NOT_FOUND, "Error with code '%s' was not found: ['args=[%s]']"},
	ERR_JSON_MARSHAL_ERROR:                 {ERR_JSON_MARSHAL_ERROR, "Error marshal JSON"},
	ERR_JSON_UNMARSHAL_ERROR:               {ERR_JSON_UNMARSHAL_ERROR, "Error unmarshal JSON"},
	ERR_XML_MARSHAL_ERROR:                  {ERR_XML_MARSHAL_ERROR, "Error marshal XMl"},
	ERR_XML_UNMARSHAL_ERROR:                {ERR_XML_UNMARSHAL_ERROR, "Error unmarshal XML"},
	ERR_YAML_MARSHAL_ERROR:                 {ERR_YAML_MARSHAL_ERROR, "Error marshal YAML"},
	ERR_YAML_UNMARSHAL_ERROR:               {ERR_YAML_UNMARSHAL_ERROR, "Error unmarshal YAML"},
	ERR_HTTP_LISTENER_CREATE_ERROR:         {ERR_HTTP_LISTENER_CREATE_ERROR, "Failed to create new TCP listener: ['network=['tcp']', 'address=[%s]']"},
	ERR_DB_CONNECTION_ERROR:                {ERR_DB_CONNECTION_ERROR, "Failed to connect DB server: ['host=[%s]', 'port=[%s]', 'dbname=[%s]', 'sslmode=[%s]', 'user=[%s]']"},
	ERR_DB_CLOSE_ERROR:                     {ERR_DB_CLOSE_ERROR, "Failed to close DB server: ['host=[%s]', 'port=[%s]', 'dbname=[%s]', 'sslmode=[%s]', 'user=[%s]']"},
	ERR_DB_TX_NOT_DEFINED_ERROR:            {ERR_DB_TX_NOT_DEFINED_ERROR, "Transaction was not defined"},
	ERR_DB_SQL_STM_NOT_DEFINED_ERROR:       {ERR_DB_SQL_STM_NOT_DEFINED_ERROR, "SQL statement was not defined: ['sql=[%s]']"},
	ERR_DB_SQL_STM_PREPARE_ERROR:           {ERR_DB_SQL_STM_PREPARE_ERROR, "Error prepare SQL statement: ['sql=[%s]']"},
	ERR_DB_TX_COMMIT_ERROR:                 {ERR_DB_TX_COMMIT_ERROR, "Error commit transaction"},
	ERR_DB_TX_ROLLBAK_ERROR:                {ERR_DB_TX_ROLLBAK_ERROR, "Error rollback transaction"},
	ERR_DB_TX_CREATE_ERROR:                 {ERR_DB_TX_CREATE_ERROR, "Error create new transaction"},
	ERR_DB_SELECT_ERROR:                    {ERR_DB_SELECT_ERROR, "Error Select SQL statement: ['sqlID=[%v]', 'sql=[%s]', 'args=[%s]']"},
	ERR_DB_GET_ERROR:                       {ERR_DB_GET_ERROR, "Error Get SQL statement: ['sqlID=[%v]', 'sql=[%s]', 'args=[%s]']"},
	ERR_DB_EXEC_ERROR:                      {ERR_DB_EXEC_ERROR, "Error Exec SQL statement: ['sqlID=[%v]', 'sql=[%s]', 'args=[%s]']"},
	ERR_DB_QUERYROWX_ERROR:                 {ERR_DB_QUERYROWX_ERROR, "Error Exec SQL statement: ['sqlID=[%v]', 'sql=[%s]', 'args=[%s]']"},
	ERR_DB_EXEC_ROW_COUNT_ERROR:            {ERR_DB_EXEC_ROW_COUNT_ERROR, "Error count exec affected rows: ['sqlID=[%v]', 'sql=[%s]', 'args=[%s]']"},
	ERR_DB_TOO_MANY_ROWS_ERROR:             {ERR_DB_TOO_MANY_ROWS_ERROR, "Too many rows found: ['args=[%s]']"},
	ERR_PANIC_RECOVER_ERROR:                {ERR_PANIC_RECOVER_ERROR, "Recover from panic: ['caller=[%s]', 'mes=[%s]', 'args=[%s]']"},
	ERR_WORKER_POOL_TIMEOUT_ERROR:          {ERR_WORKER_POOL_TIMEOUT_ERROR, "Worker pool - time for task process exceeded: ['taskID=[%v]', 'timeout=[%s]']"},
	ERR_WORKER_POOL_WORKER_ERROR:           {ERR_WORKER_POOL_WORKER_ERROR, "Worker pool - worker error: ['message=[%s]']"},
	ERR_WORKER_POOL_TASK_COUNT_ERROR:       {ERR_WORKER_POOL_TASK_COUNT_ERROR, "Worker pool service - exceeded number of Task Count: ['taskName=[%s]', 'planTaskCount=[%v]', doneChTaskCount=[%v]']"},
	ERR_WORKER_POOL_IS_SHUTTING_DOWN:       {ERR_WORKER_POOL_IS_SHUTTING_DOWN, "Worker pool shutting down in process - it is forbidden to add new tasks"},
	ERR_WORKER_POOL_TASK_CHANNEL_CLOSED:    {ERR_WORKER_POOL_TASK_CHANNEL_CLOSED, "Worker pool task channel was closed - it is forbidden to add new tasks"},
	ERR_TIMEOUT_ERROR:                      {ERR_TIMEOUT_ERROR, "Time for process exceeded: ['timeout=[%s]', 'mes=[%s]']"},
	ERR_INCORRECT_ARG_NUM_ERROR:            {ERR_INCORRECT_ARG_NUM_ERROR, "Incorrect number of arguments: ['expectedArgsNum=[%v]', 'args=[%s]']"},
	ERR_HTTP_SERVER_SHUTDOWN_ERROR:         {ERR_HTTP_SERVER_SHUTDOWN_ERROR, "Failed to shutdown HTTP server: ['errors=[%s]']"},
	ERR_HTTP_DUMP_REQUEST_ERROR:            {ERR_HTTP_DUMP_REQUEST_ERROR, "Failed to dump HTTP Request"},
	ERR_HTTP_BODY_READ_ERROR:               {ERR_HTTP_BODY_READ_ERROR, "Failed to read HTTP body"},
	ERR_HTTP_REQUEST_WRITE_ERROR:           {ERR_HTTP_REQUEST_WRITE_ERROR, "Failed to write HTTP response"},
	ERR_HTTP_METHOD_NOT_ALLOWED_ERROR:      {ERR_HTTP_METHOD_NOT_ALLOWED_ERROR, "HTTP method '%s' is not allowed: ['expectedMethod=[%s]']"},
	ERR_HTTP_AUTH_BASIC_NOT_SET_ERROR:      {ERR_HTTP_AUTH_BASIC_NOT_SET_ERROR, "Header 'Authorization' is not set"},
	ERR_HTTP_AUTH_BASIC_ERROR:              {ERR_HTTP_AUTH_BASIC_ERROR, "HTTP Basic authentication error - invalid user or password: ['user=[%s]']"},
	ERR_HTTP_AUTH_JWT_NOT_SET_ERROR:        {ERR_HTTP_AUTH_JWT_NOT_SET_ERROR, "JWT token does not present in Cookie. You have to authorize first"},
	ERR_HTTP_AUTH_JWT_INVALID_ERROR:        {ERR_HTTP_AUTH_JWT_INVALID_ERROR, "JWT token signature is invalid"},
	ERR_HTTP_AUTH_JWT_EXPIRED_ERROR:        {ERR_HTTP_AUTH_JWT_EXPIRED_ERROR, "JWT token expired or invalid. You have to authorize"},
	ERR_HTTP_AUTH_MSAD_INVALID_ERROR:       {ERR_HTTP_AUTH_MSAD_INVALID_ERROR, "Error MS AD authentication: ['server=[%s]', 'port=[%s]', 'baseDN=[%s]', 'security=[%s]', 'username=[%s]'"},
	ERR_HTTP_AUTH_MSAD_ERROR:               {ERR_HTTP_AUTH_MSAD_ERROR, "MS AD authentication - invalid user or password: ['server=[%s]', 'port=[%s]', 'baseDN=[%s]', 'security=[%s]', 'username=[%s]'"},
	ERR_HTTP_HEADER_INCORRECT_BOOL:         {ERR_HTTP_HEADER_INCORRECT_BOOL, "Incorrect boolean val '%s' for header '%s'"},
	ERR_INCORRECT_LOG_LEVEL:                {ERR_INCORRECT_LOG_LEVEL, "Incorrect log level '%s'. Allowed only 'DEBUG', 'INFO', 'ERROR'"},
	ERR_HTTP_HANDLER_METHOD_NOT_FOUND:      {ERR_HTTP_HANDLER_METHOD_NOT_FOUND, "HTTP handler method with name '%s' was not found"},
	ERR_HTTP_CALL_EMPTY_URL:                {ERR_HTTP_CALL_EMPTY_URL, "Empty DirectURL for HTTP client call"},
	ERR_HTTP_CALL_CREATE_CONTEXT_ERROR:     {ERR_HTTP_CALL_CREATE_CONTEXT_ERROR, "Failed to create new HTTP request"},
	ERR_HTTP_CALL_TIMEOUT_ERROR:            {ERR_HTTP_CALL_TIMEOUT_ERROR, "Failed to do HTTP request - timeout exceeded '%v'"},
	ERR_HTTP_CALL_OTHER_ERROR:              {ERR_HTTP_CALL_OTHER_ERROR, "Failed to do HTTP request - unknown error: ['errors=[%s]']"},
	ERR_HTTP_CALL_OTHER_NETURL_ERROR:       {ERR_HTTP_CALL_OTHER_NETURL_ERROR, "Failed to do HTTP request - unknown neturl.Error error: ['errors=[%s]']"},
	ERR_HTTP_CALL_READ_BODY_ERROR:          {ERR_HTTP_CALL_READ_BODY_ERROR, "Failed to read HTTP body from client response"},
	ERR_HTTP_CALL_URL_NOT_FOUND_ERROR:      {ERR_HTTP_CALL_URL_NOT_FOUND_ERROR, "URL was not found. Exceeded limit of attempts to call '%v': ['url=[%s]', 'args=[%s]']"},
	ERR_HTTP_CALL_METHOD_NOT_ALLOWED_ERROR: {ERR_HTTP_CALL_METHOD_NOT_ALLOWED_ERROR, "HTTP method '%s' is not allowed"},
	ERR_XLSX_COMMON_ERROR:                  {ERR_XLSX_COMMON_ERROR, "XLSX set cell error: ['message=[%s]', 'errors=[%s]']"},
	ERR_INCORRECT_CALL_ERROR:               {ERR_INCORRECT_CALL_ERROR, "Internal error - incorrect call: ['message=[%s]', 'args=[%s]']"},
	ERR_INCORRECT_TYPE_ERROR:               {ERR_INCORRECT_TYPE_ERROR, "Function '%s' got incorrect type argument '%s': ['gotType=[%s]', 'expectedType=[%s]']"},
	ERR_COMMON_ERROR:                       {ERR_COMMON_ERROR, "Common error: ['errors=[%s]']"},
	ERR_ERROR:                              {ERR_ERROR, "%s"},
	ERR_REFLECT_INCORRECT_TYPE:             {ERR_REFLECT_INCORRECT_TYPE, "Incorrect EntityMeta.Fields.Type: ['entity=[%s]', 'field=[%s]', 'type=[%s]']"},
	ERR_CONTEXT_CLOSED_ERROR:               {ERR_CONTEXT_CLOSED_ERROR, "Context was closed: ['message=[%s]', 'data=[%v]']"},
	ERR_VALIDATOR_COMMON_ERROR:             {ERR_VALIDATOR_COMMON_ERROR, "Validator error: ['errors=[%s]']"},
	ERR_VALIDATOR_STRUCT_ERROR:             {ERR_VALIDATOR_STRUCT_ERROR, "Error validate data: ['errors=[%s]', val=[%s]]"},
	ERR_VALIDATOR_MDG_ERROR:                {ERR_VALIDATOR_MDG_ERROR, "Error validate object '%s' with code '%s': [val=[%s], 'errors=[%s]]"},
	ERR_FILE_OPEN_ERROR:                    {ERR_FILE_OPEN_ERROR, "Error open file '%s': ['errors=[%s]']"},
	ERR_FILE_SYNC_ERROR:                    {ERR_FILE_SYNC_ERROR, "Error sync file '%s': ['errors=[%s]']"},
	ERR_TIME_FORMAT_ERROR:                  {ERR_TIME_FORMAT_ERROR, "'%s' has incorrect or empty date/time format '%s': ['expectedFormat=[%s]', 'errors=[%s]']"},
	ERR_INT_PARSE_ERROR:                    {ERR_INT_PARSE_ERROR, "'%s' has incorrect integer format '%s'"},
	ERR_INCORRECT_AUTH_TYPE:                {ERR_INCORRECT_AUTH_TYPE, "Incorrect YAML parameter 'http_service'.'auth_type' '%s'. Allowed only 'NONE', 'INTERNAL', 'MSAD'"},
	ERR_INCORRECT_ENTITY_SOURCE:            {ERR_INCORRECT_ENTITY_SOURCE, "Incorrect YAML parameter 'cache_service'.'entity_source' '%s' for entity '%s'. Allowed only 'DB', 'MDG', 'XLSX', 'UMDAP'"},
	ERR_EMPTY_JWT_KEY:                      {ERR_EMPTY_JWT_KEY, "JSON web token secret key is null"},
	ERR_EMPTY_HTTP_USER:                    {ERR_EMPTY_HTTP_USER, "HTTP User is null for internal authentication"},
	ERR_EMPTY_CONFIG_FILE:                  {ERR_EMPTY_CONFIG_FILE, "Config file is null"},
	ERR_CONFIG_FILE_NOT_EXISTS:             {ERR_CONFIG_FILE_NOT_EXISTS, "Config file '%s' does not exist"},
	ERR_CONFIG_FILE_LOAD_ERROR:             {ERR_CONFIG_FILE_LOAD_ERROR, "Error load config file '%s'"},
}
