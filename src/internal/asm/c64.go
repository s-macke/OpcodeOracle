package asm

// LookupC64Hardware returns the hardware register name for an address,
// handling I/O register mirroring. Returns empty string if not a known register.
func LookupC64Hardware(addr uint16) (name string, ok bool) {
	// Normalize mirrored addresses to canonical form
	canonical := addr
	switch {
	case addr >= 0xD000 && addr <= 0xD3FF:
		// VIC-II mirrors every 64 bytes
		canonical = 0xD000 + (addr & 0x3F)
	case addr >= 0xD400 && addr <= 0xD7FF:
		// SID mirrors every 32 bytes
		canonical = 0xD400 + (addr & 0x1F)
	case addr >= 0xDC00 && addr <= 0xDCFF:
		// CIA1 mirrors every 16 bytes
		canonical = 0xDC00 + (addr & 0x0F)
	case addr >= 0xDD00 && addr <= 0xDDFF:
		// CIA2 mirrors every 16 bytes
		canonical = 0xDD00 + (addr & 0x0F)
	}
	name, ok = C64Hardware[canonical]
	return
}

// C64Hardware maps addresses to C64 hardware register names.
// Full coverage of VIC-II, SID, CIA1, CIA2, and hardware vectors.
var C64Hardware = map[uint16]string{
	// VIC-II Sprite positions ($D000-$D00F)
	0xD000: "VIC_SPR0_X",
	0xD001: "VIC_SPR0_Y",
	0xD002: "VIC_SPR1_X",
	0xD003: "VIC_SPR1_Y",
	0xD004: "VIC_SPR2_X",
	0xD005: "VIC_SPR2_Y",
	0xD006: "VIC_SPR3_X",
	0xD007: "VIC_SPR3_Y",
	0xD008: "VIC_SPR4_X",
	0xD009: "VIC_SPR4_Y",
	0xD00A: "VIC_SPR5_X",
	0xD00B: "VIC_SPR5_Y",
	0xD00C: "VIC_SPR6_X",
	0xD00D: "VIC_SPR6_Y",
	0xD00E: "VIC_SPR7_X",
	0xD00F: "VIC_SPR7_Y",

	// VIC-II Control registers ($D010-$D02E)
	0xD010: "VIC_SPR_XMSB",
	0xD011: "VIC_CTRL1",
	0xD012: "VIC_RASTER",
	0xD013: "VIC_LPEN_X",
	0xD014: "VIC_LPEN_Y",
	0xD015: "VIC_SPR_ENA",
	0xD016: "VIC_CTRL2",
	0xD017: "VIC_SPR_YEXP",
	0xD018: "VIC_MEMPTR",
	0xD019: "VIC_IRQ",
	0xD01A: "VIC_IRQEN",
	0xD01B: "VIC_SPR_PRIO",
	0xD01C: "VIC_SPR_MC",
	0xD01D: "VIC_SPR_XEXP",
	0xD01E: "VIC_SPR_COLL",
	0xD01F: "VIC_SPR_BGCOLL",
	0xD020: "VIC_BORDER",
	0xD021: "VIC_BG0",
	0xD022: "VIC_BG1",
	0xD023: "VIC_BG2",
	0xD024: "VIC_BG3",
	0xD025: "VIC_SPR_MC0",
	0xD026: "VIC_SPR_MC1",
	0xD027: "VIC_SPR0_COL",
	0xD028: "VIC_SPR1_COL",
	0xD029: "VIC_SPR2_COL",
	0xD02A: "VIC_SPR3_COL",
	0xD02B: "VIC_SPR4_COL",
	0xD02C: "VIC_SPR5_COL",
	0xD02D: "VIC_SPR6_COL",
	0xD02E: "VIC_SPR7_COL",

	// SID Voice 1 ($D400-$D406)
	0xD400: "SID_V1_FREQ_LO",
	0xD401: "SID_V1_FREQ_HI",
	0xD402: "SID_V1_PW_LO",
	0xD403: "SID_V1_PW_HI",
	0xD404: "SID_V1_CTRL",
	0xD405: "SID_V1_AD",
	0xD406: "SID_V1_SR",

	// SID Voice 2 ($D407-$D40D)
	0xD407: "SID_V2_FREQ_LO",
	0xD408: "SID_V2_FREQ_HI",
	0xD409: "SID_V2_PW_LO",
	0xD40A: "SID_V2_PW_HI",
	0xD40B: "SID_V2_CTRL",
	0xD40C: "SID_V2_AD",
	0xD40D: "SID_V2_SR",

	// SID Voice 3 ($D40E-$D414)
	0xD40E: "SID_V3_FREQ_LO",
	0xD40F: "SID_V3_FREQ_HI",
	0xD410: "SID_V3_PW_LO",
	0xD411: "SID_V3_PW_HI",
	0xD412: "SID_V3_CTRL",
	0xD413: "SID_V3_AD",
	0xD414: "SID_V3_SR",

	// SID Filter/Volume ($D415-$D418)
	0xD415: "SID_FC_LO",
	0xD416: "SID_FC_HI",
	0xD417: "SID_RES_FILT",
	0xD418: "SID_VOLUME",

	// SID Misc ($D419-$D41C)
	0xD419: "SID_POTX",
	0xD41A: "SID_POTY",
	0xD41B: "SID_OSC3",
	0xD41C: "SID_ENV3",

	// CIA1 ($DC00-$DC0F)
	0xDC00: "CIA1_PRA",
	0xDC01: "CIA1_PRB",
	0xDC02: "CIA1_DDRA",
	0xDC03: "CIA1_DDRB",
	0xDC04: "CIA1_TALO",
	0xDC05: "CIA1_TAHI",
	0xDC06: "CIA1_TBLO",
	0xDC07: "CIA1_TBHI",
	0xDC08: "CIA1_TOD_10",
	0xDC09: "CIA1_TOD_SEC",
	0xDC0A: "CIA1_TOD_MIN",
	0xDC0B: "CIA1_TOD_HR",
	0xDC0C: "CIA1_SDR",
	0xDC0D: "CIA1_ICR",
	0xDC0E: "CIA1_CRA",
	0xDC0F: "CIA1_CRB",

	// CIA2 ($DD00-$DD0F)
	0xDD00: "CIA2_PRA",
	0xDD01: "CIA2_PRB",
	0xDD02: "CIA2_DDRA",
	0xDD03: "CIA2_DDRB",
	0xDD04: "CIA2_TALO",
	0xDD05: "CIA2_TAHI",
	0xDD06: "CIA2_TBLO",
	0xDD07: "CIA2_TBHI",
	0xDD08: "CIA2_TOD_10",
	0xDD09: "CIA2_TOD_SEC",
	0xDD0A: "CIA2_TOD_MIN",
	0xDD0B: "CIA2_TOD_HR",
	0xDD0C: "CIA2_SDR",
	0xDD0D: "CIA2_ICR",
	0xDD0E: "CIA2_CRA",
	0xDD0F: "CIA2_CRB",

	// Hardware vectors ($FFFA-$FFFF)
	0xFFFA: "VECTOR_NMI_LO",
	0xFFFB: "VECTOR_NMI_HI",
	0xFFFC: "VECTOR_RESET_LO",
	0xFFFD: "VECTOR_RESET_HI",
	0xFFFE: "VECTOR_IRQ_LO",
	0xFFFF: "VECTOR_IRQ_HI",
}
