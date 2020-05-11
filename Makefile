lndb: $(shell find . -name "*.go") assets.gen.go
	go build -ldflags="-s -w" -o ./lndb

assets.gen.go: $(shell find public -type f)
	broccoli -src=public/ -o assets

deploy: lndb
	ssh root@nusakan-58 'systemctl stop lndb'
	scp lndb nusakan-58:lndb/lndb
	ssh root@nusakan-58 'systemctl start lndb'
