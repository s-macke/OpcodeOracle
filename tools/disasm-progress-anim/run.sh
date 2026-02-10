  set -e
  go run . \
  --opcodeoracle-bin ../../opcodeoracle \
  --input-glob '../../testdata/archive/*.opcodeoracle.json' \
  --output ../..//disasm_progress_go_tools_only.mp4
