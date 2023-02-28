go tool pprof goprofex http://127.0.0.1:3001/debug/pprof/profile
rem go tool pprof goprofex http://127.0.0.1:3000/debug/pprof/goroutine
rem go tool pprof -alloc_objects goprofex http://127.0.0.1:3000/debug/pprof/heap

rem web
rem top100 -cum (sync|runtime)