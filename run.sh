set -e

(cd src && go build -o ../ ./cmd/opcodeoracle)

./opcodeoracle new prg testdata/weltendaemmerung.bin --entry 2061

#0x0801
#2016
