../ab.exe -T application/json -l -k -c 1 -n 100000 "http://127.0.0.1:3001/app/system/entities/Currency/json" > dbselect_0001_parallel.txt
../ab.exe -T application/json -l -k -c 8 -n 100000 "http://127.0.0.1:3001/app/system/entities/Currency/json" > dbselect_0008_parallel.txt