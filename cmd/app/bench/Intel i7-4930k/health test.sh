../ab -T application/json -l -k -c 1 -n 100000 "http://127.0.0.1:3001/app/system/health" > health_0001_parallel.txt
../ab -T application/json -l -k -c 8 -n 100000 "http://127.0.0.1:3001/app/system/health" > health_0008_parallel.txt
