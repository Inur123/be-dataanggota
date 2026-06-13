# Makefile untuk DataAnggota Backend

.PHONY: run build dev tidy migrate deploy

# Run server (development)
run:
	go run ./cmd/server

# Build binary locally
build:
	go build -o bin/anggota-backend ./cmd/server

# Clean binary
clean:
	rm -rf bin/

# Compile for Linux (GCP VPS)
build-linux:
	CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o bin/anggota-backend ./cmd/server

# Deploy otomatis ke VPS
# Perintah ini mematikan service di VPS, meng-upload binary baru, dan menghidupkannya kembali
deploy: build-linux
	@echo "====== DEPLOYING DATAANGGOTA BACKEND TO VPS ======"
	ssh devzainur@34.50.73.15 "sudo systemctl stop dataanggota-backend"
	scp bin/anggota-backend devzainur@34.50.73.15:/var/www/be-dataanggota/anggota-backend
	ssh devzainur@34.50.73.15 "sudo chmod +x /var/www/be-dataanggota/anggota-backend && sudo systemctl start dataanggota-backend"
	@echo "====== DEPLOYMENT BERHASIL! ======"
