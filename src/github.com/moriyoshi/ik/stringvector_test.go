package ik

import "testing"

func Append(t *testing.T) {
	sv := make(StringVector, 0)
	if len(sv) != 0 { t.Fail() }
	if cap(sv) != 0 { t.Fail() }
	for i := 0; i < 512; i += 3 {
		if i > 1 && sv[i - 1] != "c" { t.Fail() }
		sv.Append("a")
		if len(sv) != i + 1 { t.Fail() }
		if i < 253 { if cap(sv) != i + 1 { t.Fail() } }
		if sv[i] != "a" { t.Fail() }
		sv.Append("b")
		if len(sv) != i + 2 { t.Fail() }
		if i < 253 { if cap(sv) != i + 2 { t.Fail() } }
		if sv[i] != "a" { t.Fail() }
		if sv[i + 1] != "b" { t.Fail() }
		sv.Append("c")
		if len(sv) != i + 3 { t.Fail() }
		if i < 253 { if cap(sv) != i + 3 { t.Fail() } }
		if sv[i] != "a" { t.Fail() }
		if sv[i + 1] != "b" { t.Fail() }
		if sv[i + 2] != "c" { t.Fail() }
	}
}

func TestStringVector_Pop(t *testing.T) {
	sv := make(StringVector, 2)
	sv[0] = "a"
	sv[1] = "b"
	if len(sv) != 2 { t.Fail() }
	if cap(sv) != 2 { t.Fail() }
	if sv.Pop() != "b" { t.Fail() }
	if len(sv) != 1 { t.Fail() }
	if cap(sv) != 2 { t.Fail() }
	if sv.Pop() != "a" { t.Fail() }
	if len(sv) != 0 { t.Fail() }
	if cap(sv) != 2 { t.Fail() }
}

func TestStringVector_Shift(t *testing.T) {
	sv := make(StringVector, 2)
	sv[0] = "a"
	sv[1] = "b"
	if len(sv) != 2 { t.Fail() }
	if cap(sv) != 2 { t.Fail() }
	if sv.Shift() != "a" { t.Fail() }
	if len(sv) != 1 { t.Fail() }
	if cap(sv) != 1 { t.Fail() }
	if sv.Shift() != "b" { t.Fail() }
	if len(sv) != 0 { t.Fail() }
	if cap(sv) != 0 { t.Fail() }
}
