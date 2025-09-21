run:
	go run cmd/main.go

server:
	go run cmd/server/main.go -path . -addr :8080

ui:
	cd mind-map-react && npm run dev

ui-build:
	cd mind-map-react && npm run build
