package matasano

import "testing"

func TestXor(t *testing.T) {
	in1 := []byte("1c0111001f010100061a024b53535009181c")
	in2 := []byte("686974207468652062756c6c277320657965")
	expected := "746865206b696420646f6e277420706c6179"
	if expected != string(Xor(in1, in2)) {
		t.Fail()
	}
}
