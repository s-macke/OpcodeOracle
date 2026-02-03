set -e

(cd src && go build -o ../ ./cmd/opcodeoracle)

#./opcodeoracle new prg --entry 2061 testdata/weltendaemmerung.prg

./opcodeoracle new sid testdata/Nippon.sid
#./opcodeoracle validate testdata/Nippon.opcodeoracle.json
#./opcodeoracle disasm --start 0x1000 --end 0x1100 testdata/Nippon.opcodeoracle.json

#./opcodeoracle edit annotation --address 0xC000 --extend --comment "This is an annotation" testdata/Nippon.opcodeoracle.json
#./opcodeoracle edit headline --address 0xC000 --extend --comment "This is a headline" testdata/Nippon.opcodeoracle.json

#./opcodeoracle disasm --start 0xC001 --end 0xC003 testdata/Nippon.opcodeoracle.json


#./opcodeoracle disasm testdata/Nippon.opcodeoracle.json

#./opcodeoracle edit annotation --address 0xC041 --type inline --comment "This is a header1" testdata/Nippon.opcodeoracle.json
#./opcodeoracle edit annotation --address 0xC041 --type inline --extend --comment "This is a header2" testdata/Nippon.opcodeoracle.json

#./opcodeoracle edit annotation --address 0xC040 --type inline --comment "This is a header2" testdata/Nippon.opcodeoracle.json
#./opcodeoracle edit annotation --address 0xC040 --type headline --comment "This is a header3" testdata/Nippon.opcodeoracle.json
#./opcodeoracle edit annotation --address 0xC03E --type headline --comment "This is a header3" testdata/Nippon.opcodeoracle.json

#./opcodeoracle edit symbol --address 0xC03E --type word --name "VAR_1" testdata/Nippon.opcodeoracle.json
./opcodeoracle edit symbol --address 0xC0BC --type word --name "VAR_1" testdata/Nippon.opcodeoracle.json

./opcodeoracle export testdata/Nippon.opcodeoracle.json
