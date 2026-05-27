cd admin/web
npm install
npm run build
cd ../..

go build -o styxpress-admin ./cmd/styxpress-admin
./styxpress-admin