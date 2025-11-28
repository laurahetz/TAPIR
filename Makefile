build:
	podman build -f container/Containerfile . -t tapir

run: 
	podman run -v ./app/:/usr/local/go/scr/tapir/app/configs --name apir_dpf tapir /usr/local/go/scr/tapir/bench --path=/usr/local/go/scr/tapir/app/configs/configs.json --out=/usr/local/go/scr/tapir/app/configs/results.csv

apir_dpf:
	podman run -v ./app/:/usr/local/go/scr/tapir/app/configs --name apir_dpf tapir /usr/local/go/scr/tapir/bench --path=/usr/local/go/scr/tapir/app/configs/configs_apir_dpf.json --out=/usr/local/go/scr/tapir/app/configs/results_apir_dpf.csv

apir_matrix_pp:
	podman run -v ./app/:/usr/local/go/scr/tapir/app/configs --name apir_matrix_pp tapir /usr/local/go/scr/tapir/bench --path=/usr/local/go/scr/tapir/app/configs/configs_apir_matrix_pp.json --out=/usr/local/go/scr/tapir/app/configs/results_apir_matrix_pp.csv

apir_matrix_mt:
	podman run -v ./app/:/usr/local/go/scr/tapir/app/configs --name apir_matrix_mt tapir /usr/local/go/scr/tapir/bench --path=/usr/local/go/scr/tapir/app/configs/configs_apir_matrix_mt.json --out=/usr/local/go/scr/tapir/app/configs/results_apir_matrix_mt.csv

singlepass:
	podman run -v ./app/:/usr/local/go/scr/tapir/app/configs --name singlepass tapir /usr/local/go/scr/tapir/bench --path=/usr/local/go/scr/tapir/app/configs/configs_singlepass.json --out=/usr/local/go/scr/tapir/app/configs/results_singlepass.csv

tapir_pp:
	podman run -v ./app/:/usr/local/go/scr/tapir/app/configs --name tapir_pp tapir /usr/local/go/scr/tapir/bench --path=/usr/local/go/scr/tapir/app/configs/configs_tapir_pp.json --out=/usr/local/go/scr/tapir/app/configs/results_tapir_pp.csv

tapir_mt:
	podman run -v ./app/:/usr/local/go/scr/tapir/app/configs --name tapir_mt tapir /usr/local/go/scr/tapir/bench --path=/usr/local/go/scr/tapir/app/configs/configs_tapir_mt.json --out=/usr/local/go/scr/tapir/app/configs/results_tapir_mt.csv


rm_all:
	podman rm -f run apir_dpf singlepass apir_matrix_pp apir_matrix_mt tapir_pp tapir_mt