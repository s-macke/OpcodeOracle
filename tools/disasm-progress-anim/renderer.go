package main

import (
	"errors"
	"flag"
	"fmt"
	"image"
	"image/color"
	"image/png"
	"math"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"
)

var (
	hexColorRe = regexp.MustCompile(`^#[0-9a-fA-F]{6}$`)
)

type Config struct {
	InputGlob       string
	OpcodeOracleBin string
	Output          string
	Width           int
	Height          int
	Seconds         float64
	FPS             int
	MaxFrames       int
	Columns         int
	MinColumns      int
	MaxColumns      int
	ColumnGap       int

	BGColor      string
	DotColor     string
	AddressColor string
	CommentColor string
	TextColor    string
	OpcodeColor  string

	GlowEnabled                 bool
	GlowColor                   string
	GlowDecaySeconds            float64
	GlowRadius                  float64
	GlowIntensity               float64
	GlowSpreadPx                int
	GlowThresholdMinPixelChange int
	DotRadius                   int
	FFmpegCRF                   int
	FFmpegPreset                string
	KeepTempFrames              bool
	AAEnabled                   bool
	AAScale                     int
}

type lineLayout struct {
	cols        int
	rowsPerCol  int
	startX      int
	startY      int
	columnWidth float64
	rowHeight   float64
	charWidth   float64
	columnGap   int
	maxLineLen  int
}

type point struct {
	x int
	y int
}

type rgb struct {
	r uint8
	g uint8
	b uint8
}

type diffTag uint8

const (
	diffEqual diffTag = iota
	diffInsert
	diffDelete
	diffReplace
)

type lineOp struct {
	tag                diffTag
	prevStart, prevEnd int
	currStart, currEnd int
}

const (
	kindAddress = iota
	kindComment
	kindOpcode
	kindText
	kindCount
)

var kindNames = [kindCount]string{"address", "comment", "opcode", "text"}

func ParseConfig(args []string) (Config, error) {
	cfg := Config{}
	fs := flag.NewFlagSet("disasm-progress-anim", flag.ContinueOnError)
	fs.SetOutput(os.Stderr)

	fs.StringVar(&cfg.InputGlob, "input-glob", "testdata/archive/*.opcodeoracle.json", "Glob pattern for input state files.")
	fs.StringVar(&cfg.OpcodeOracleBin, "opcodeoracle-bin", "./opcodeoracle", "Path to opcodeoracle executable.")
	fs.StringVar(&cfg.Output, "output", "testdata/archive/disasm_progress_960x540.mp4", "Output MP4 file path.")
	fs.IntVar(&cfg.Width, "width", 960, "Output video width.")
	fs.IntVar(&cfg.Height, "height", 540, "Output video height.")
	fs.Float64Var(&cfg.Seconds, "seconds", 12.0, "Target video duration in seconds.")
	fs.IntVar(&cfg.FPS, "fps", 30, "Output frames per second.")
	fs.IntVar(&cfg.MaxFrames, "max-frames", 0, "Optional hard cap for frame count (0 means auto from seconds*fps).")
	fs.IntVar(&cfg.Columns, "columns", 0, "Fixed number of line columns. 0 = auto-select.")
	fs.IntVar(&cfg.MinColumns, "min-columns", 5, "Minimum auto-selected columns.")
	fs.IntVar(&cfg.MaxColumns, "max-columns", 8, "Maximum auto-selected columns.")
	fs.IntVar(&cfg.ColumnGap, "column-gap", 10, "Pixel gap between columns.")

	fs.StringVar(&cfg.BGColor, "bg-color", "#0D0A07", "Background color as hex, e.g. #0D0A07.")
	fs.StringVar(&cfg.DotColor, "dot-color", "#FFB347", "Deprecated alias for --address-color.")
	fs.StringVar(&cfg.AddressColor, "address-color", "", "Color for $xx/$xxxx tokens as hex (default: same as --dot-color).")
	fs.StringVar(&cfg.CommentColor, "comment-color", "#5A5A5A", "Color for comment text (; ...) as hex.")
	fs.StringVar(&cfg.TextColor, "text-color", "#C8C8C8", "Color for all other non-whitespace text as hex.")
	fs.StringVar(&cfg.OpcodeColor, "opcode-color", "#8FD3FF", "Color for leading 3-letter opcodes (pattern: four spaces + AAA + space).")

	cfg.GlowEnabled = true
	fs.BoolVar(&cfg.GlowEnabled, "glow-enabled", true, "Enable green glow for newly added dots.")
	fs.StringVar(&cfg.GlowColor, "glow-color", "#69FF8E", "Glow color as hex.")
	fs.Float64Var(&cfg.GlowDecaySeconds, "glow-decay-seconds", 0.25, "Glow decay time constant in seconds.")
	fs.Float64Var(&cfg.GlowRadius, "glow-radius", 2.0, "Glow blur radius in pixels.")
	fs.Float64Var(&cfg.GlowIntensity, "glow-intensity", 0.6, "Glow alpha multiplier in range [0,1].")
	fs.IntVar(&cfg.GlowSpreadPx, "glow-spread-px", 2, "Pixel spread before blur.")
	fs.IntVar(&cfg.GlowThresholdMinPixelChange, "glow-threshold-min-pixel-change", 1, "Minimum newly added dots required to trigger glow injection.")

	fs.IntVar(&cfg.DotRadius, "dot-radius", 1, "Dot radius in pixels (1 = single pixel dots).")
	fs.IntVar(&cfg.FFmpegCRF, "ffmpeg-crf", 34, "Initial x264 CRF value.")
	fs.StringVar(&cfg.FFmpegPreset, "ffmpeg-preset", "slow", "x264 preset.")
	fs.BoolVar(&cfg.KeepTempFrames, "keep-temp-frames", false, "Keep temporary PNG frames directory.")

	cfg.AAEnabled = true
	fs.BoolVar(&cfg.AAEnabled, "aa-enabled", true, "Enable supersampled antialiasing.")
	fs.IntVar(&cfg.AAScale, "aa-scale", 2, "Supersampling scale factor (1 or 2 recommended).")

	filteredArgs := make([]string, 0, len(args))
	for _, a := range args {
		switch a {
		case "--no-glow-enabled":
			cfg.GlowEnabled = false
		case "--no-aa-enabled":
			cfg.AAEnabled = false
		default:
			filteredArgs = append(filteredArgs, a)
		}
	}

	if err := fs.Parse(filteredArgs); err != nil {
		return Config{}, err
	}

	if cfg.AddressColor == "" {
		cfg.AddressColor = cfg.DotColor
	}

	if err := validateConfig(cfg); err != nil {
		return Config{}, err
	}

	return cfg, nil
}

func validateConfig(cfg Config) error {
	if cfg.Width <= 0 || cfg.Height <= 0 {
		return errors.New("width/height must be positive integers")
	}
	if cfg.Seconds <= 0 {
		return errors.New("seconds must be positive")
	}
	if cfg.FPS <= 0 {
		return errors.New("fps must be positive")
	}
	if cfg.DotRadius < 0 {
		return errors.New("dot-radius must be zero or positive")
	}
	if cfg.Columns < 0 {
		return errors.New("columns must be >= 0")
	}
	if cfg.MinColumns <= 0 || cfg.MaxColumns <= 0 {
		return errors.New("min/max columns must be positive")
	}
	if cfg.MinColumns > cfg.MaxColumns {
		return errors.New("min-columns must be <= max-columns")
	}
	if cfg.ColumnGap < 0 {
		return errors.New("column-gap must be >= 0")
	}
	if cfg.GlowDecaySeconds <= 0 {
		return errors.New("glow-decay-seconds must be > 0")
	}
	if cfg.GlowRadius < 0 {
		return errors.New("glow-radius must be >= 0")
	}
	if cfg.GlowIntensity < 0 || cfg.GlowIntensity > 1 {
		return errors.New("glow-intensity must be in range [0, 1]")
	}
	if cfg.GlowSpreadPx < 0 {
		return errors.New("glow-spread-px must be >= 0")
	}
	if cfg.GlowThresholdMinPixelChange < 1 {
		return errors.New("glow-threshold-min-pixel-change must be >= 1")
	}
	if cfg.AAScale < 1 {
		return errors.New("aa-scale must be >= 1")
	}
	if cfg.FFmpegCRF < 0 || cfg.FFmpegCRF > 51 {
		return errors.New("ffmpeg-crf must be in range [0, 51]")
	}
	colors := []string{cfg.BGColor, cfg.AddressColor, cfg.CommentColor, cfg.TextColor, cfg.OpcodeColor, cfg.GlowColor}
	for _, c := range colors {
		if !hexColorRe.MatchString(c) {
			return fmt.Errorf("invalid color '%s' (expected #RRGGBB)", c)
		}
	}
	return nil
}

func parseHexColor(value string) rgb {
	r, _ := strconv.ParseUint(value[1:3], 16, 8)
	g, _ := strconv.ParseUint(value[3:5], 16, 8)
	b, _ := strconv.ParseUint(value[5:7], 16, 8)
	return rgb{uint8(r), uint8(g), uint8(b)}
}

func Run(cfg Config) error {
	bgRGB := parseHexColor(cfg.BGColor)
	addressRGB := parseHexColor(cfg.AddressColor)
	commentRGB := parseHexColor(cfg.CommentColor)
	opcodeRGB := parseHexColor(cfg.OpcodeColor)
	textRGB := parseHexColor(cfg.TextColor)
	glowRGB := parseHexColor(cfg.GlowColor)

	files, err := findInputs(cfg.InputGlob)
	if err != nil {
		return err
	}

	aaScale := 1
	if cfg.AAEnabled {
		aaScale = cfg.AAScale
	}
	renderWidth := cfg.Width * aaScale
	renderHeight := cfg.Height * aaScale

	autoFrames := max(2, int(math.Round(cfg.Seconds*float64(cfg.FPS))))
	wantedFrames := autoFrames
	if cfg.MaxFrames > 0 {
		wantedFrames = cfg.MaxFrames
	}
	indices := sampledIndices(len(files), wantedFrames)
	selected := make([]string, 0, len(indices))
	for _, idx := range indices {
		selected = append(selected, files[idx])
	}

	allLines := make([][]string, 0, len(selected))
	for i, p := range selected {
		fmt.Fprintf(os.Stderr, "[%d/%d] disasm %s\n", i+1, len(selected), filepath.Base(p))
		lines, derr := disasmLinesFromState(cfg.OpcodeOracleBin, p)
		if derr != nil {
			return derr
		}
		allLines = append(allLines, lines)
	}

	maxLines := 1
	maxLineLen := 1
	for _, lines := range allLines {
		if len(lines) > maxLines {
			maxLines = len(lines)
		}
		for _, line := range lines {
			if len(line) > maxLineLen {
				maxLineLen = len(line)
			}
		}
	}

	layout, err := buildLayout(renderWidth, renderHeight, maxLines, maxLineLen, cfg.MinColumns, cfg.MaxColumns, cfg.ColumnGap, cfg.Columns)
	if err != nil {
		return err
	}

	fmt.Fprintf(
		os.Stderr,
		"[config] output=%s size=%dx%d render_size=%dx%d fps=%d aa_scale=%d\n",
		cfg.Output, cfg.Width, cfg.Height, renderWidth, renderHeight, cfg.FPS, aaScale,
	)

	if err := os.MkdirAll(filepath.Dir(cfg.Output), 0o755); err != nil {
		return fmt.Errorf("creating output directory: %w", err)
	}

	tmpDir := ""
	if cfg.KeepTempFrames {
		tmpDir, err = os.MkdirTemp("", "disasm_progress_frames_")
		if err != nil {
			return fmt.Errorf("creating temp dir: %w", err)
		}
	} else {
		tmpDir, err = os.MkdirTemp("", "disasm_progress_frames_")
		if err != nil {
			return fmt.Errorf("creating temp dir: %w", err)
		}
		defer func() {
			_ = os.RemoveAll(tmpDir)
			fmt.Fprintln(os.Stderr, "[cleanup] temp frames removed")
		}()
	}
	fmt.Fprintf(os.Stderr, "[render] temp_frames_dir=%s\n", tmpDir)

	var prevLines []string
	glowState := map[point]float64{}
	decayFactor := math.Exp(-(1.0 / float64(cfg.FPS)) / cfg.GlowDecaySeconds)

	for i, lines := range allLines {
		frameIdx := i + 1
		fmt.Fprintf(os.Stderr, "[render %d/%d] build points\n", frameIdx, len(allLines))

		progress := 0.0
		if len(allLines) > 1 {
			progress = float64(i) / float64(len(allLines)-1)
		}

		pointsByKind := collectPointsByKind(lines, layout, renderWidth, renderHeight)
		newPoints := map[point]struct{}{}
		if i > 0 {
			newPoints = semanticNewPoints(prevLines, lines, layout, renderWidth, renderHeight)
		}

		fmt.Fprintf(
			os.Stderr,
			"[render %d/%d] points text=%d comment=%d opcode=%d address=%d new_glow_points=%d\n",
			frameIdx,
			len(allLines),
			len(pointsByKind[kindText]),
			len(pointsByKind[kindComment]),
			len(pointsByKind[kindOpcode]),
			len(pointsByKind[kindAddress]),
			len(newPoints),
		)

		frame := paintFrame(pointsByKind, renderWidth, renderHeight, bgRGB, addressRGB, commentRGB, opcodeRGB, textRGB, cfg.DotRadius*aaScale, progress)

		if cfg.GlowEnabled {
			updateGlowState(glowState, newPoints, decayFactor, cfg.GlowThresholdMinPixelChange)
			addGlowOverlay(frame, glowState, glowRGB, cfg.GlowSpreadPx*aaScale, cfg.GlowRadius*float64(aaScale), cfg.GlowIntensity)
			fmt.Fprintf(os.Stderr, "[render %d/%d] glow_active=%d\n", frameIdx, len(allLines), len(glowState))
		}

		if aaScale > 1 {
			fmt.Fprintf(os.Stderr, "[render %d/%d] downsample box\n", frameIdx, len(allLines))
			frame = downsampleBox(frame, aaScale)
		}

		framePath := filepath.Join(tmpDir, fmt.Sprintf("frame_%05d.png", i))
		if err := writePNG(framePath, frame); err != nil {
			return err
		}
		fmt.Fprintf(os.Stderr, "[render %d/%d] saved %s\n", frameIdx, len(allLines), framePath)
		prevLines = lines
	}

	framePattern := filepath.Join(tmpDir, "frame_%05d.png")
	fmt.Fprintf(os.Stderr, "[encode] start ffmpeg pattern=%s\n", framePattern)
	if err := encodeMP4(framePattern, cfg.FPS, cfg.Output, cfg.FFmpegCRF, cfg.FFmpegPreset); err != nil {
		return err
	}
	fmt.Fprintf(os.Stderr, "[encode] finished output=%s\n", cfg.Output)

	st, err := os.Stat(cfg.Output)
	if err != nil {
		return fmt.Errorf("stat output: %w", err)
	}
	sizeBytes := st.Size()
	if sizeBytes > 3*1024*1024 {
		fmt.Fprintf(os.Stderr, "[encode] size=%d > 3MB, re-encoding with higher CRF\n", sizeBytes)
		if err := encodeMP4(framePattern, cfg.FPS, cfg.Output, min(51, cfg.FFmpegCRF+3), cfg.FFmpegPreset); err != nil {
			return err
		}
		st2, err := os.Stat(cfg.Output)
		if err != nil {
			return fmt.Errorf("stat output after second pass: %w", err)
		}
		sizeBytes = st2.Size()
		fmt.Fprintf(os.Stderr, "[encode] second pass finished size=%d\n", sizeBytes)
	}

	fmt.Printf("output: %s\n", cfg.Output)
	fmt.Printf("frames: %d @ %d fps\n", len(allLines), cfg.FPS)
	fmt.Printf("duration: %.2fs\n", float64(len(allLines))/float64(cfg.FPS))
	fmt.Printf("layout: %d columns, %d rows/column, max_line_len=%d\n", layout.cols, layout.rowsPerCol, layout.maxLineLen)
	fmt.Printf("colors: address=%s, comment=%s, opcode=%s, text=%s\n", cfg.AddressColor, cfg.CommentColor, cfg.OpcodeColor, cfg.TextColor)
	fmt.Printf(
		"glow: enabled=%t, color=%s, decay=%gs, spread=%dpx, radius=%g, intensity=%g\n",
		cfg.GlowEnabled,
		cfg.GlowColor,
		cfg.GlowDecaySeconds,
		cfg.GlowSpreadPx,
		cfg.GlowRadius,
		cfg.GlowIntensity,
	)
	fmt.Printf("aa: enabled=%t, scale=%d\n", cfg.AAEnabled, aaScale)
	fmt.Printf("size: %.2f MB\n", float64(sizeBytes)/(1024*1024))
	if cfg.KeepTempFrames {
		fmt.Printf("frames_dir: %s\n", tmpDir)
	}

	return nil
}

func findInputs(pattern string) ([]string, error) {
	files, err := filepath.Glob(pattern)
	if err != nil {
		return nil, fmt.Errorf("invalid input glob: %w", err)
	}
	out := make([]string, 0, len(files))
	for _, f := range files {
		if strings.HasSuffix(f, ".opcodeoracle.json") {
			out = append(out, f)
		}
	}
	sort.Strings(out)
	if len(out) == 0 {
		return nil, fmt.Errorf("no input files found for glob: %s", pattern)
	}
	if len(out) < 2 {
		return nil, errors.New("at least two input files are required to animate progress")
	}
	return out, nil
}

func sampledIndices(total int, wanted int) []int {
	if wanted >= total {
		idxs := make([]int, total)
		for i := range total {
			idxs[i] = i
		}
		return idxs
	}
	if wanted <= 1 {
		return []int{total - 1}
	}
	idxs := make([]int, 0, wanted)
	for i := range wanted {
		pos := int(math.Round(float64(i) * float64(total-1) / float64(wanted-1)))
		if len(idxs) == 0 || idxs[len(idxs)-1] != pos {
			idxs = append(idxs, pos)
		}
	}
	return idxs
}

func disasmLinesFromState(opcodeoracleBin string, stateFile string) ([]string, error) {
	cmd := exec.Command(opcodeoracleBin, "disasm", stateFile)
	out, err := cmd.Output()
	if err != nil {
		if ee, ok := err.(*exec.ExitError); ok {
			return nil, fmt.Errorf(
				"disasm failed for %s\ncommand: %s disasm %s\nstderr:\n%s",
				stateFile,
				opcodeoracleBin,
				stateFile,
				strings.TrimSpace(string(ee.Stderr)),
			)
		}
		return nil, fmt.Errorf("failed to run opcodeoracle binary '%s': %w", opcodeoracleBin, err)
	}
	output := string(out)
	norm := strings.ReplaceAll(output, "\r\n", "\n")
	norm = strings.ReplaceAll(norm, "\r", "\n")
	return strings.Split(strings.TrimRight(norm, "\n"), "\n"), nil
}

func chooseColumns(maxLines int, maxLineLen int, drawableWidth int, drawableHeight int, minCols int, maxCols int, colGap int, fixedCols int) int {
	if fixedCols > 0 {
		return fixedCols
	}
	bestCols := minCols
	bestScore := math.Inf(-1)
	for cols := minCols; cols <= maxCols; cols++ {
		rows := int(math.Ceil(float64(maxLines) / float64(cols)))
		totalGap := colGap * (cols - 1)
		availW := drawableWidth - totalGap
		if availW <= cols {
			continue
		}
		colW := float64(availW) / float64(cols)
		rowH := float64(drawableHeight) / float64(rows)
		charW := colW / float64(maxLineLen)
		score := minFloat(rowH, charW*2.2)
		if score > bestScore {
			bestScore = score
			bestCols = cols
		}
	}
	return bestCols
}

func buildLayout(width int, height int, maxLines int, maxLineLen int, minCols int, maxCols int, colGap int, fixedCols int) (lineLayout, error) {
	marginX := max(16, int(float64(width)*0.03))
	marginTop := max(12, int(float64(height)*0.03))
	marginBottom := max(32, int(float64(height)*0.08))
	drawableW := width - 2*marginX
	drawableH := height - marginTop - marginBottom
	if drawableW <= 0 || drawableH <= 0 {
		return lineLayout{}, errors.New("invalid width/height: drawable area is not positive")
	}

	cols := chooseColumns(maxLines, maxLineLen, drawableW, drawableH, minCols, maxCols, colGap, fixedCols)
	rows := int(math.Ceil(float64(maxLines) / float64(cols)))
	availW := drawableW - colGap*(cols-1)
	if availW <= 0 {
		return lineLayout{}, errors.New("column-gap too large for drawable width")
	}

	colW := float64(availW) / float64(cols)
	rowH := float64(drawableH) / float64(rows)
	charW := colW / float64(maxLineLen)

	return lineLayout{
		cols:        cols,
		rowsPerCol:  rows,
		startX:      marginX,
		startY:      marginTop,
		columnWidth: colW,
		rowHeight:   rowH,
		charWidth:   charW,
		columnGap:   colGap,
		maxLineLen:  maxLineLen,
	}, nil
}

func collectPointsByKind(lines []string, layout lineLayout, width int, height int) [kindCount][]point {
	var byKind [kindCount][]point
	maxLines := layout.cols * layout.rowsPerCol

	for lineIdx, line := range lines {
		if lineIdx >= maxLines {
			break
		}
		col := lineIdx / layout.rowsPerCol
		row := lineIdx % layout.rowsPerCol
		if col >= layout.cols {
			break
		}
		colStart := int(float64(layout.startX) + float64(col)*(layout.columnWidth+float64(layout.columnGap)))
		y := int(float64(layout.startY) + float64(row)*layout.rowHeight)
		if y < 0 || y >= height {
			continue
		}

		trimmed := line
		if len(trimmed) > layout.maxLineLen {
			trimmed = trimmed[:layout.maxLineLen]
		}

		commentIdx := strings.IndexByte(trimmed, ';')
		codePart := trimmed
		if commentIdx >= 0 {
			codePart = trimmed[:commentIdx]
		}

		addrPos := addressCharPositions(codePart)
		opcodePos := opcodeCharPositions(codePart)
		lastX := [kindCount]int{-1, -1, -1, -1}

		for charIdx := 0; charIdx < len(trimmed); charIdx++ {
			ch := trimmed[charIdx]
			if ch == ' ' || ch == '\t' {
				continue
			}
			x := int(float64(colStart) + float64(charIdx)*layout.charWidth)
			kind := kindText
			switch {
			case commentIdx >= 0 && charIdx >= commentIdx:
				kind = kindComment
			case opcodePos[charIdx]:
				kind = kindOpcode
			case addrPos[charIdx]:
				kind = kindAddress
			default:
				kind = kindText
			}
			if x == lastX[kind] {
				continue
			}
			if x >= 0 && x < width {
				byKind[kind] = append(byKind[kind], point{x: x, y: y})
				lastX[kind] = x
			}
		}
	}

	return byKind
}

func addressCharPositions(codePart string) map[int]bool {
	out := map[int]bool{}
	n := len(codePart)
	isHex := func(b byte) bool {
		return (b >= '0' && b <= '9') || (b >= 'A' && b <= 'F') || (b >= 'a' && b <= 'f')
	}

	for i := 0; i < n; i++ {
		if codePart[i] != '$' {
			continue
		}
		if i > 0 && isHex(codePart[i-1]) {
			continue
		}
		matched := 0
		for _, digits := range []int{4, 2} {
			end := i + 1 + digits
			if end > n {
				continue
			}
			ok := true
			for j := i + 1; j < end; j++ {
				if !isHex(codePart[j]) {
					ok = false
					break
				}
			}
			if !ok {
				continue
			}
			if end < n && isHex(codePart[end]) {
				continue
			}
			matched = digits
			break
		}
		if matched > 0 {
			for j := i; j < i+1+matched; j++ {
				if codePart[j] != ' ' && codePart[j] != '\t' {
					out[j] = true
				}
			}
			i += matched
		}
	}
	return out
}

func opcodeCharPositions(codePart string) map[int]bool {
	out := map[int]bool{}
	if len(codePart) < 8 {
		return out
	}
	if !strings.HasPrefix(codePart, "    ") || codePart[7] != ' ' {
		return out
	}
	for i := 4; i <= 6; i++ {
		if codePart[i] < 'A' || codePart[i] > 'Z' {
			return out
		}
	}
	out[4], out[5], out[6] = true, true, true
	return out
}

func semanticNewPoints(prevLines []string, currLines []string, layout lineLayout, width int, height int) map[point]struct{} {
	out := map[point]struct{}{}
	maxVisible := layout.cols * layout.rowsPerCol
	prev := prevLines
	curr := currLines
	if len(prev) > maxVisible {
		prev = prev[:maxVisible]
	}
	if len(curr) > maxVisible {
		curr = curr[:maxVisible]
	}
	ops := lineDiffOps(prev, curr)
	for _, op := range ops {
		switch op.tag {
		case diffEqual, diffDelete:
			continue
		case diffInsert:
			for lineIdx := op.currStart; lineIdx < op.currEnd; lineIdx++ {
				for _, charIdx := range nonWhitespaceCharPositions(curr[lineIdx], layout.maxLineLen) {
					if p, ok := lineCharToPoint(lineIdx, charIdx, layout, width, height); ok {
						out[p] = struct{}{}
					}
				}
			}
		case diffReplace:
			prevLen := op.prevEnd - op.prevStart
			currLen := op.currEnd - op.currStart
			common := min(prevLen, currLen)
			for k := 0; k < common; k++ {
				prevLine := prev[op.prevStart+k]
				lineIdx := op.currStart + k
				currLine := curr[lineIdx]
				for _, charIdx := range changedCharPositions(prevLine, currLine, layout.maxLineLen) {
					if p, ok := lineCharToPoint(lineIdx, charIdx, layout, width, height); ok {
						out[p] = struct{}{}
					}
				}
			}
			for lineIdx := op.currStart + common; lineIdx < op.currEnd; lineIdx++ {
				for _, charIdx := range nonWhitespaceCharPositions(curr[lineIdx], layout.maxLineLen) {
					if p, ok := lineCharToPoint(lineIdx, charIdx, layout, width, height); ok {
						out[p] = struct{}{}
					}
				}
			}
		}
	}

	return out
}

func lineDiffOps(prev, curr []string) []lineOp {
	n := len(prev)
	m := len(curr)
	lcs := make([][]int, n+1)
	for i := range lcs {
		lcs[i] = make([]int, m+1)
	}
	for i := n - 1; i >= 0; i-- {
		for j := m - 1; j >= 0; j-- {
			if prev[i] == curr[j] {
				lcs[i][j] = lcs[i+1][j+1] + 1
			} else {
				lcs[i][j] = max(lcs[i+1][j], lcs[i][j+1])
			}
		}
	}

	i, j := 0, 0
	ops := make([]lineOp, 0, 16)
	for i < n || j < m {
		if i < n && j < m && prev[i] == curr[j] {
			appendLineOp(&ops, diffEqual, i, i+1, j, j+1)
			i++
			j++
			continue
		}
		if j < m && (i == n || lcs[i][j+1] >= lcs[i+1][j]) {
			appendLineOp(&ops, diffInsert, i, i, j, j+1)
			j++
		} else if i < n {
			appendLineOp(&ops, diffDelete, i, i+1, j, j)
			i++
		}
	}
	return coalesceReplaceOps(ops)
}

func appendLineOp(ops *[]lineOp, tag diffTag, ps, pe, cs, ce int) {
	if len(*ops) == 0 {
		*ops = append(*ops, lineOp{tag: tag, prevStart: ps, prevEnd: pe, currStart: cs, currEnd: ce})
		return
	}
	last := &(*ops)[len(*ops)-1]
	if last.tag == tag && last.prevEnd == ps && last.currEnd == cs {
		last.prevEnd = pe
		last.currEnd = ce
		return
	}
	*ops = append(*ops, lineOp{tag: tag, prevStart: ps, prevEnd: pe, currStart: cs, currEnd: ce})
}

func coalesceReplaceOps(ops []lineOp) []lineOp {
	out := make([]lineOp, 0, len(ops))
	for i := 0; i < len(ops); {
		op := ops[i]
		if op.tag == diffDelete {
			j := i
			delStart := ops[j].prevStart
			delEnd := ops[j].prevEnd
			currPos := ops[j].currStart
			for j+1 < len(ops) && ops[j+1].tag == diffDelete && ops[j+1].currStart == currPos {
				j++
				delEnd = ops[j].prevEnd
			}
			if j+1 < len(ops) && ops[j+1].tag == diffInsert && ops[j+1].prevStart == delEnd {
				ins := ops[j+1]
				out = append(out, lineOp{
					tag:       diffReplace,
					prevStart: delStart,
					prevEnd:   delEnd,
					currStart: ins.currStart,
					currEnd:   ins.currEnd,
				})
				i = j + 2
				continue
			}
		}
		if op.tag == diffInsert {
			j := i
			insStart := ops[j].currStart
			insEnd := ops[j].currEnd
			prevPos := ops[j].prevStart
			for j+1 < len(ops) && ops[j+1].tag == diffInsert && ops[j+1].prevStart == prevPos {
				j++
				insEnd = ops[j].currEnd
			}
			if j+1 < len(ops) && ops[j+1].tag == diffDelete && ops[j+1].currStart == insEnd {
				del := ops[j+1]
				out = append(out, lineOp{
					tag:       diffReplace,
					prevStart: del.prevStart,
					prevEnd:   del.prevEnd,
					currStart: insStart,
					currEnd:   insEnd,
				})
				i = j + 2
				continue
			}
		}
		out = append(out, op)
		i++
	}
	return out
}

func changedCharPositions(prevLine string, currLine string, maxLineLen int) []int {
	limA := min(len(prevLine), maxLineLen)
	limB := min(len(currLine), maxLineLen)
	lim := max(limA, limB)
	out := make([]int, 0, 16)
	for i := 0; i < lim; i++ {
		a := byte(' ')
		b := byte(' ')
		if i < limA {
			a = prevLine[i]
		}
		if i < limB {
			b = currLine[i]
		}
		if a == b {
			continue
		}
		if b == ' ' || b == '\t' {
			continue
		}
		out = append(out, i)
	}
	return out
}

func nonWhitespaceCharPositions(line string, maxLineLen int) []int {
	lim := min(len(line), maxLineLen)
	out := make([]int, 0, lim)
	for i := 0; i < lim; i++ {
		if line[i] != ' ' && line[i] != '\t' {
			out = append(out, i)
		}
	}
	return out
}

func lineCharToPoint(lineIdx int, charIdx int, layout lineLayout, width int, height int) (point, bool) {
	if lineIdx < 0 || charIdx < 0 || charIdx >= layout.maxLineLen {
		return point{}, false
	}
	col := lineIdx / layout.rowsPerCol
	row := lineIdx % layout.rowsPerCol
	if col >= layout.cols {
		return point{}, false
	}
	colStart := int(float64(layout.startX) + float64(col)*(layout.columnWidth+float64(layout.columnGap)))
	x := int(float64(colStart) + float64(charIdx)*layout.charWidth)
	y := int(float64(layout.startY) + float64(row)*layout.rowHeight)
	if x < 0 || x >= width || y < 0 || y >= height {
		return point{}, false
	}
	return point{x: x, y: y}, true
}

func paintFrame(
	pointsByKind [kindCount][]point,
	width int,
	height int,
	bgRGB rgb,
	addressRGB rgb,
	commentRGB rgb,
	opcodeRGB rgb,
	textRGB rgb,
	dotRadius int,
	progress float64,
) *image.RGBA {
	frame := image.NewRGBA(image.Rect(0, 0, width, height))
	fillRect(frame, image.Rect(0, 0, width, height), color.RGBA{R: bgRGB.r, G: bgRGB.g, B: bgRGB.b, A: 255})

	setPoints(frame, pointsByKind[kindText], textRGB, dotRadius)
	setPoints(frame, pointsByKind[kindComment], commentRGB, dotRadius)
	setPoints(frame, pointsByKind[kindOpcode], opcodeRGB, dotRadius)
	setPoints(frame, pointsByKind[kindAddress], addressRGB, dotRadius)

	barW := int(float64(width) * 0.84)
	barH := max(4, height/120)
	barX := (width - barW) / 2
	barY := height - max(16, height/30)
	fillRect(frame, image.Rect(barX, barY-barH, barX+barW, barY), color.RGBA{R: 35, G: 30, B: 24, A: 255})
	fillW := int(float64(barW) * clamp(progress, 0, 1))
	fillRect(frame, image.Rect(barX, barY-barH, barX+fillW, barY), color.RGBA{R: 170, G: 120, B: 60, A: 255})

	return frame
}

func setPoints(img *image.RGBA, pts []point, c rgb, dotRadius int) {
	if dotRadius <= 1 {
		for _, p := range pts {
			if image.Pt(p.x, p.y).In(img.Bounds()) {
				o := img.PixOffset(p.x, p.y)
				img.Pix[o+0] = c.r
				img.Pix[o+1] = c.g
				img.Pix[o+2] = c.b
				img.Pix[o+3] = 255
			}
		}
		return
	}

	for _, p := range pts {
		for dy := -dotRadius; dy <= dotRadius; dy++ {
			y := p.y + dy
			if y < 0 || y >= img.Bounds().Dy() {
				continue
			}
			for dx := -dotRadius; dx <= dotRadius; dx++ {
				x := p.x + dx
				if x < 0 || x >= img.Bounds().Dx() {
					continue
				}
				o := img.PixOffset(x, y)
				img.Pix[o+0] = c.r
				img.Pix[o+1] = c.g
				img.Pix[o+2] = c.b
				img.Pix[o+3] = 255
			}
		}
	}
}

func updateGlowState(glowState map[point]float64, newPoints map[point]struct{}, decayFactor float64, minNewPoints int) {
	for p, v := range glowState {
		nv := v * decayFactor
		if nv < 0.01 {
			delete(glowState, p)
		} else {
			glowState[p] = nv
		}
	}
	if len(newPoints) < minNewPoints {
		return
	}
	for p := range newPoints {
		glowState[p] = 1.0
	}
}

func addGlowOverlay(frame *image.RGBA, glowState map[point]float64, glowRGB rgb, glowSpreadPx int, glowRadius float64, glowIntensity float64) {
	if len(glowState) == 0 {
		return
	}
	w := frame.Bounds().Dx()
	h := frame.Bounds().Dy()
	mask := make([]uint8, w*h)

	stampRadius := max(0, glowSpreadPx)
	for p, v := range glowState {
		alpha := uint8(clamp(v, 0, 1) * 255)
		if stampRadius == 0 {
			if p.x >= 0 && p.x < w && p.y >= 0 && p.y < h {
				idx := p.y*w + p.x
				if alpha > mask[idx] {
					mask[idx] = alpha
				}
			}
			continue
		}
		for dy := -stampRadius; dy <= stampRadius; dy++ {
			y := p.y + dy
			if y < 0 || y >= h {
				continue
			}
			for dx := -stampRadius; dx <= stampRadius; dx++ {
				x := p.x + dx
				if x < 0 || x >= w {
					continue
				}
				idx := y*w + x
				if alpha > mask[idx] {
					mask[idx] = alpha
				}
			}
		}
	}

	if glowRadius > 0 {
		r := max(1, int(math.Round(glowRadius)))
		mask = boxBlurAlpha(mask, w, h, r)
	}

	if glowIntensity < 1 {
		for i := range mask {
			mask[i] = uint8(float64(mask[i]) * glowIntensity)
		}
	}

	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			idx := y*w + x
			a := mask[idx]
			if a == 0 {
				continue
			}
			o := frame.PixOffset(x, y)
			addR := (int(glowRGB.r) * int(a)) / 255
			addG := (int(glowRGB.g) * int(a)) / 255
			addB := (int(glowRGB.b) * int(a)) / 255
			frame.Pix[o+0] = uint8(min(255, int(frame.Pix[o+0])+addR))
			frame.Pix[o+1] = uint8(min(255, int(frame.Pix[o+1])+addG))
			frame.Pix[o+2] = uint8(min(255, int(frame.Pix[o+2])+addB))
			frame.Pix[o+3] = 255
		}
	}
}

func boxBlurAlpha(src []uint8, w int, h int, r int) []uint8 {
	if r <= 0 {
		dst := make([]uint8, len(src))
		copy(dst, src)
		return dst
	}
	tmp := make([]uint8, len(src))
	dst := make([]uint8, len(src))
	window := 2*r + 1

	for y := 0; y < h; y++ {
		sum := 0
		for x := -r; x <= r; x++ {
			xx := clampInt(x, 0, w-1)
			sum += int(src[y*w+xx])
		}
		for x := 0; x < w; x++ {
			tmp[y*w+x] = uint8(sum / window)
			left := clampInt(x-r, 0, w-1)
			right := clampInt(x+r+1, 0, w-1)
			sum += int(src[y*w+right]) - int(src[y*w+left])
		}
	}

	for x := 0; x < w; x++ {
		sum := 0
		for y := -r; y <= r; y++ {
			yy := clampInt(y, 0, h-1)
			sum += int(tmp[yy*w+x])
		}
		for y := 0; y < h; y++ {
			dst[y*w+x] = uint8(sum / window)
			top := clampInt(y-r, 0, h-1)
			bot := clampInt(y+r+1, 0, h-1)
			sum += int(tmp[bot*w+x]) - int(tmp[top*w+x])
		}
	}

	return dst
}

func downsampleBox(src *image.RGBA, scale int) *image.RGBA {
	if scale <= 1 {
		return src
	}
	sw := src.Bounds().Dx()
	sh := src.Bounds().Dy()
	dw := sw / scale
	dh := sh / scale
	if dw <= 0 || dh <= 0 {
		return src
	}
	dst := image.NewRGBA(image.Rect(0, 0, dw, dh))
	area := scale * scale

	for y := 0; y < dh; y++ {
		for x := 0; x < dw; x++ {
			sr, sg, sb := 0, 0, 0
			for by := 0; by < scale; by++ {
				sy := y*scale + by
				for bx := 0; bx < scale; bx++ {
					sx := x*scale + bx
					o := src.PixOffset(sx, sy)
					sr += int(src.Pix[o+0])
					sg += int(src.Pix[o+1])
					sb += int(src.Pix[o+2])
				}
			}
			do := dst.PixOffset(x, y)
			dst.Pix[do+0] = uint8(sr / area)
			dst.Pix[do+1] = uint8(sg / area)
			dst.Pix[do+2] = uint8(sb / area)
			dst.Pix[do+3] = 255
		}
	}
	return dst
}

func writePNG(path string, img image.Image) error {
	f, err := os.Create(path)
	if err != nil {
		return fmt.Errorf("creating frame '%s': %w", path, err)
	}
	defer f.Close()
	enc := png.Encoder{CompressionLevel: png.BestSpeed}
	if err := enc.Encode(f, img); err != nil {
		return fmt.Errorf("writing frame '%s': %w", path, err)
	}
	return nil
}

func encodeMP4(framePattern string, fps int, outputPath string, crf int, preset string) error {
	ffmpegBin, err := exec.LookPath("ffmpeg")
	if err != nil {
		return errors.New("ffmpeg not found in PATH")
	}
	cmd := exec.Command(
		ffmpegBin,
		"-y",
		"-loglevel", "error",
		"-framerate", strconv.Itoa(fps),
		"-i", framePattern,
		"-c:v", "libx264",
		"-pix_fmt", "yuv420p",
		"-crf", strconv.Itoa(crf),
		"-preset", preset,
		"-movflags", "+faststart",
		outputPath,
	)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("ffmpeg encoding failed for output '%s': %w", outputPath, err)
	}
	return nil
}

func fillRect(img *image.RGBA, rect image.Rectangle, c color.RGBA) {
	r := rect.Intersect(img.Bounds())
	if r.Empty() {
		return
	}
	for y := r.Min.Y; y < r.Max.Y; y++ {
		for x := r.Min.X; x < r.Max.X; x++ {
			o := img.PixOffset(x, y)
			img.Pix[o+0] = c.R
			img.Pix[o+1] = c.G
			img.Pix[o+2] = c.B
			img.Pix[o+3] = c.A
		}
	}
}

func clamp(v, lo, hi float64) float64 {
	if v < lo {
		return lo
	}
	if v > hi {
		return hi
	}
	return v
}

func clampInt(v, lo, hi int) int {
	if v < lo {
		return lo
	}
	if v > hi {
		return hi
	}
	return v
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}

func minFloat(a, b float64) float64 {
	if a < b {
		return a
	}
	return b
}
