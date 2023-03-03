../ab.exe -T application/json -p wp_factorial.json -l -k -c 1 -n 100000 "http://127.0.0.1:3001/app/system/wp_factorial?wp_tipe=bg" > wp_factorial_bg_0001_parallel.txt
../ab.exe -T application/json -p wp_factorial.json -l -k -c 8 -n 1000000 "http://127.0.0.1:3001/app/system/wp_factorial?wp_tipe=bg" > wp_factorial_bg_0008_parallel.txt
../ab.exe -T application/json -p wp_factorial.json -l -k -c 32 -n 1000000 "http://127.0.0.1:3001/app/system/wp_factorial?wp_tipe=bg" > wp_factorial_bg_0032_parallel.txt

../ab.exe -T application/json -p wp_factorial.json -l -k -c 1 -n 100000 "http://127.0.0.1:3001/app/system/wp_factorial?wp_tipe=online" > wp_factorial_online_0001_parallel.txt
../ab.exe -T application/json -p wp_factorial.json -l -k -c 8 -n 1000000 "http://127.0.0.1:3001/app/system/wp_factorial?wp_tipe=online" > wp_factorial_online_0008_parallel.txt
../ab.exe -T application/json -p wp_factorial.json -l -k -c 32 -n 1000000 "http://127.0.0.1:3001/app/system/wp_factorial?wp_tipe=online" > wp_factorial_online_0032_parallel.txt


#../ab.exe -T application/json -p wp_factorial.json -l -k -c 1 -n 100000 "http://127.0.0.1:3001/app/system/wp_factorial?wp_tipe=online" > wp_factorial_online_0001_parallel.txt
#../ab.exe -T application/json -p wp_factorial.json -l -k -c 8 -n 100000 "http://127.0.0.1:3001/app/system/wp_factorial?wp_tipe=online" > wp_factorial_online_0008_parallel.txt