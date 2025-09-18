generate-mindmap:
	go run cmd/main.go

run:
	go run cmd/server/main.go

web:
	cd mind-map-react && npm start