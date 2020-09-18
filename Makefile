microdb: $(shell find . -name "*.go") assets.gen.go
	go build -ldflags="-s -w" -o ./microdb

assets.gen.go: $(shell find public -type f)
	broccoli -src=public/ -o assets

deploy: microdb
	ssh root@nusakan-58 'systemctl stop microdb'
	scp microdb nusakan-58:microdb/microdb
	ssh root@nusakan-58 'systemctl start microdb'
