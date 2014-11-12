package ik

import "testing"

func TestIntVector_Append(t *testing.T) {
	sv := make(IntVector, 0)
	if len(sv) != 0 { t.Fail() }
	if cap(sv) != 0 { t.Fail() }
	for i := 0; i < 512; i += 3 {
		if i > 1 && sv[i - 1] != 19 { t.Fail() }
		sv.Append(11)
		if len(sv) != i + 1 { t.Fail() }
		if i < 253 { if cap(sv) != i + 1 { t.Fail() } }
		if sv[i] != 11 { t.Fail() }
		sv.Append(17)
		if len(sv) != i + 2 { t.Fail() }
		if i < 253 { if cap(sv) != i + 2 { t.Fail() } }
		if sv[i] != 11 { t.Fail() }
		if sv[i + 1] != 17 { t.Fail() }
		sv.Append(19)
		if len(sv) != i + 3 { t.Fail() }
		if i < 253 { if cap(sv) != i + 3 { t.Fail() } }
		if sv[i] != 11 { t.Fail() }
		if sv[i + 1] != 17 { t.Fail() }
		if sv[i + 2] != 19 { t.Fail() }
	}
}

func TestIntVector_Pop(t *testing.T) {
	sv := make(IntVector, 2)
	sv[0] = 11
	sv[1] = 17
	if len(sv) != 2 { t.Fail() }
	if cap(sv) != 2 { t.Fail() }
	if sv.Pop() != 17 { t.Fail() }
	if len(sv) != 1 { t.Fail() }
	if cap(sv) != 2 { t.Fail() }
	if sv.Pop() != 11 { t.Fail() }
	if len(sv) != 0 { t.Fail() }
	if cap(sv) != 2 { t.Fail() }
}

func TestIntVector_Shift(t *testing.T) {
	sv := make(IntVector, 2)
	sv[0] = 11
	sv[1] = 17
	if len(sv) != 2 { t.Fail() }
	if cap(sv) != 2 { t.Fail() }
	if sv.Shift() != 11 { t.Fail() }
	if len(sv) != 1 { t.Fail() }
	if cap(sv) != 1 { t.Fail() }
	if sv.Shift() != 17 { t.Fail() }
	if len(sv) != 0 { t.Fail() }
	if cap(sv) != 0 { t.Fail() }
}
