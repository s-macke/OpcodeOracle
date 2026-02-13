package main

import "testing"

func TestLineDiffOpsInsertionNoShift(t *testing.T) {
	prev := []string{"A", "B", "C", "D"}
	curr := []string{"A", "B", "X", "C", "D"}

	ops := lineDiffOps(prev, curr)
	if len(ops) != 3 {
		t.Fatalf("expected 3 ops, got %d: %#v", len(ops), ops)
	}
	if ops[0].tag != diffEqual || ops[0].prevStart != 0 || ops[0].prevEnd != 2 || ops[0].currStart != 0 || ops[0].currEnd != 2 {
		t.Fatalf("unexpected op0: %#v", ops[0])
	}
	if ops[1].tag != diffInsert || ops[1].prevStart != 2 || ops[1].prevEnd != 2 || ops[1].currStart != 2 || ops[1].currEnd != 3 {
		t.Fatalf("unexpected op1: %#v", ops[1])
	}
	if ops[2].tag != diffEqual || ops[2].prevStart != 2 || ops[2].prevEnd != 4 || ops[2].currStart != 3 || ops[2].currEnd != 5 {
		t.Fatalf("unexpected op2: %#v", ops[2])
	}
}

func TestSemanticNewPointsInsertionOnlyNewLineGlows(t *testing.T) {
	layout := lineLayout{
		cols:        1,
		rowsPerCol:  16,
		startX:      0,
		startY:      0,
		columnWidth: 200,
		rowHeight:   10,
		charWidth:   1,
		columnGap:   0,
		maxLineLen:  80,
	}

	prev := []string{"A", "B", "C", "D"}
	curr := []string{"A", "B", "X", "C", "D"}

	pts := semanticNewPoints(prev, curr, layout, 400, 200)
	if len(pts) != 1 {
		t.Fatalf("expected exactly 1 new point for inserted single-char line, got %d", len(pts))
	}
	if _, ok := pts[point{x: 0, y: 20}]; !ok {
		t.Fatalf("expected inserted line point at y=20, got %#v", pts)
	}
	if _, bad := pts[point{x: 0, y: 30}]; bad {
		t.Fatalf("unexpected glow on shifted unchanged line C at y=30")
	}
	if _, bad := pts[point{x: 0, y: 40}]; bad {
		t.Fatalf("unexpected glow on shifted unchanged line D at y=40")
	}
}

func TestLineDiffOpsReplace(t *testing.T) {
	prev := []string{"A", "B", "C"}
	curr := []string{"A", "X", "C"}
	ops := lineDiffOps(prev, curr)
	if len(ops) != 3 {
		t.Fatalf("expected 3 ops, got %d: %#v", len(ops), ops)
	}
	if ops[1].tag != diffReplace {
		t.Fatalf("expected middle op replace, got %#v", ops[1])
	}
}
