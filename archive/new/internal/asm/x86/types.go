package x86

type Mnemonic string

type Prefix string

const (
	PrefixES    Prefix = "es"
	PrefixCS    Prefix = "cs"
	PrefixSS    Prefix = "ss"
	PrefixDS    Prefix = "ds"
	PrefixFS    Prefix = "fs"
	PrefixGS    Prefix = "gs"
	PrefixLock  Prefix = "lock"
	PrefixRepZ  Prefix = "repz"
	PrefixRepNZ Prefix = "repnz"
	PrefixOp32  Prefix = "32"
)

type AddressingKind string

const (
	AddressingNone      AddressingKind = "none"
	AddressingRegister  AddressingKind = "register"
	AddressingImmediate AddressingKind = "immediate"
	AddressingMemory    AddressingKind = "memory"
	AddressingRelative  AddressingKind = "relative"
	AddressingMixed     AddressingKind = "mixed"
)

type FlowKind string

const (
	FlowNone            FlowKind = "none"
	FlowJump            FlowKind = "jump"
	FlowCall            FlowKind = "call"
	FlowReturn          FlowKind = "return"
	FlowInterrupt       FlowKind = "interrupt"
	FlowConditionalJump FlowKind = "conditional_jump"
)

type OperandKind string

const (
	OperandRegister  OperandKind = "register"
	OperandImmediate OperandKind = "immediate"
	OperandMemory    OperandKind = "memory"
	OperandRelative  OperandKind = "relative"
	OperandFar       OperandKind = "far"
	OperandText      OperandKind = "text"
)

type OperandWidth string

const (
	WidthNone OperandWidth = ""
	Width8    OperandWidth = "byte"
	Width16   OperandWidth = "word"
)

type Register string

const (
	RegAL Register = "al"
	RegCL Register = "cl"
	RegDL Register = "dl"
	RegBL Register = "bl"
	RegAH Register = "ah"
	RegCH Register = "ch"
	RegDH Register = "dh"
	RegBH Register = "bh"
	RegAX Register = "ax"
	RegCX Register = "cx"
	RegDX Register = "dx"
	RegBX Register = "bx"
	RegSP Register = "sp"
	RegBP Register = "bp"
	RegSI Register = "si"
	RegDI Register = "di"
	RegES Register = "es"
	RegCS Register = "cs"
	RegSS Register = "ss"
	RegDS Register = "ds"
	RegFS Register = "fs"
	RegGS Register = "gs"
)

type BaseAddressExpr string

const (
	BaseBXSI   BaseAddressExpr = "bx+si"
	BaseBXDI   BaseAddressExpr = "bx+di"
	BaseBPSI   BaseAddressExpr = "bp+si"
	BaseBPDI   BaseAddressExpr = "bp+di"
	BaseSI     BaseAddressExpr = "si"
	BaseDI     BaseAddressExpr = "di"
	BaseBP     BaseAddressExpr = "bp"
	BaseBX     BaseAddressExpr = "bx"
	BaseDirect BaseAddressExpr = ""
)

type PointerQualifier string

const (
	PtrNone PointerQualifier = ""
	PtrByte PointerQualifier = "byte ptr"
	PtrWord PointerQualifier = "word ptr"
)

type TargetKind string

const (
	TargetNone TargetKind = "none"
	TargetNear TargetKind = "near"
	TargetFar  TargetKind = "far"
)

type RelativeRef struct {
	BaseOffset uint16
	Delta      int16
	Resolved   uint16
}

type MemoryRef struct {
	SegmentOverride *Register
	Base            BaseAddressExpr
	Displacement    int16
	DirectAddress   *uint16
	Size            OperandWidth
	Pointer         PointerQualifier
}

type Target struct {
	Kind     TargetKind
	Near     *uint16
	Far      *FarAddress
	Indirect bool
}

type Operand struct {
	Kind         OperandKind
	Width        OperandWidth
	Register     Register
	Immediate    uint16
	Displacement int16
	Memory       *MemoryRef
	Relative     *RelativeRef
	Far          *FarAddress
	Text         string
}
