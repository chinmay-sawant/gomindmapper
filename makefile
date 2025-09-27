# this will generate the functionmap.json
run:
	go run cmd/main.go -path gopdfsuit --include-external=true --skip-folders="golang.org,gin-gonic,bytedance,ugorji,go-playground"

# this will start the server 
server:
	go run cmd/server/main.go -path gopdfsuit -addr :8080 --include-external=true --skip-folders="golang.org,gin-gonic,bytedance,ugorji,go-playground"

ui:
	cd mind-map-react && npm run dev

# this will build the ui
ui-build:
	cd mind-map-react && npm run build