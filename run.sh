set -e

(cd src && go build -o ../ ./cmd/opcodeoracle)

./opcodeoracle new prg --entry 2061 testdata/weltendaemmerung.prg

./opcodeoracle new sid testdata/Nippon.sid
