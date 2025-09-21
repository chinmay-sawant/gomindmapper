run:
	go run cmd/main.go

server:
	go run cmd/server/main.go -path . -addr :8080

ui:
	cd /d "D:\Chinmay_Personal_Projects\GoMindMapper\mind-map-react" && npm run dev

ui-build:
	cd /d "D:\Chinmay_Personal_Projects\GoMindMapper\mind-map-react" && npm run build
