package aes

import "crypto/cipher"

const (
	// BlockSize is the size of each AES block
	BlockSize = 16
)

// cipherAES is a block cipher that implements the Advances Encryption Standard (AES)
type cipherAES struct {
	expkey []uint32
}

// NewCipher creates a cipherAES object
func NewCipher(key []byte) cipher.Block {
	expkey := keyExpansion(key)
	return cipherAES{expkey}
}

func (a cipherAES) Encrypt(dst, src []byte) {
	state := make([]uint32, 4)
	pack(state, src[0:BlockSize])
	encrypt(state, a.expkey)
	unpack(dst[0:BlockSize], state)
}

func (a cipherAES) Decrypt(dst, src []byte) {
	state := make([]uint32, 4)
	pack(state, src[0:BlockSize])
	decrypt(state, a.expkey)
	unpack(dst[0:BlockSize], state)
}

func (a cipherAES) BlockSize() int {
	return BlockSize
}

// decrypt decrypts the ciphertext in the state []uint32
// under the expanded key expkey
func decrypt(state, expkey []uint32) {
	keyi := len(expkey) - 4
	addRoundKey(state, expkey[keyi:keyi+4])
	keyi -= 4
	rounds := len(expkey)/4 - 2
	for i := 0; i < rounds; i++ {
		invShiftRows(state)
		invSubBytes(state)
		addRoundKey(state, expkey[keyi:keyi+4])
		keyi -= 4
		invMixColumns(state)
	}
	invShiftRows(state)
	invSubBytes(state)
	addRoundKey(state, expkey[keyi:keyi+4])
}

// encrypt encrypts the plaintext in the state []uint32
// under the expanded key expkey
func encrypt(state, expkey []uint32) {
	keyi := 0
	addRoundKey(state, expkey[keyi:keyi+4])
	keyi += 4
	rounds := len(expkey)/4 - 2
	for i := 0; i < rounds; i++ {
		subBytes(state)
		shiftRows(state)
		mixColumns(state)
		addRoundKey(state, expkey[keyi:keyi+4])
		keyi += 4
	}
	subBytes(state)
	shiftRows(state)
	addRoundKey(state, expkey[keyi:keyi+4])
}

func invShiftRows(state []uint32) {
	for i := 1; i < 4; i++ {
		state[i] = rotWordRight(state[i], uint(i))
	}
}

func shiftRows(state []uint32) {
	for i := 1; i < 4; i++ {
		state[i] = rotWordLeft(state[i], uint(i))
	}
}

func invSubBytes(state []uint32) {
	for i := range state {
		state[i] = invSubWord(state[i])
	}
}

func subBytes(state []uint32) {
	for i := range state {
		state[i] = subWord(state[i])
	}
}

func addRoundKey(state, key []uint32) {
	for i := range state {
		state[i] = state[i] ^ key[i]
	}
}

// based on https://en.wikipedia.org/wiki/Rijndael_mix_columns#InverseMixColumns
func invMixColumns(state []uint32) {
	// a0-3 represent the bytes of a column
	// r0-3 are the transformed bytes
	calcInvMixCols := func(a0, a1, a2, a3 byte) (r0, r1, r2, r3 byte) {
		r0 = gMulBy14[a0] ^ gMulBy11[a1] ^ gMulBy13[a2] ^ gMulBy9[a3] // 14*a0 + 11*a1 + 13*a2 +  9*a3
		r1 = gMulBy9[a0] ^ gMulBy14[a1] ^ gMulBy11[a2] ^ gMulBy13[a3] //  9*a0 + 14*a1 + 11*a2 + 13*a3
		r2 = gMulBy13[a0] ^ gMulBy9[a1] ^ gMulBy14[a2] ^ gMulBy11[a3] // 13*a0 +  9*a1 + 14*a2 + 11*a3
		r3 = gMulBy11[a0] ^ gMulBy13[a1] ^ gMulBy9[a2] ^ gMulBy14[a3] // 11*a0 + 13*a1 +  9*a2 + 14*a3
		return
	}
	manipulateColumns(state, calcInvMixCols)
}

// based on https://en.wikipedia.org/wiki/Rijndael_mix_columns#MixColumns
func mixColumns(state []uint32) {
	// a0-3 represent the bytes of a column
	// r0-3 are the transformed bytes
	calcMixCols := func(a0, a1, a2, a3 byte) (r0, r1, r2, r3 byte) {
		r0 = gMulBy2[a0] ^ gMulBy3[a1] ^ a2 ^ a3 // 2*a0 + 3*a1 + a2   + a3
		r1 = a0 ^ gMulBy2[a1] ^ gMulBy3[a2] ^ a3 // a0   + 2*a1 + 3*a2 + a3
		r2 = a0 ^ a1 ^ gMulBy2[a2] ^ gMulBy3[a3] // a0   + a1   + 2*a2 + 3*a3
		r3 = gMulBy3[a0] ^ a1 ^ a2 ^ gMulBy2[a3] // 3*a0 + a1   + a2   + 2*a3
		return
	}
	manipulateColumns(state, calcMixCols)
}

func manipulateColumns(state []uint32, calc func(byte, byte, byte, byte) (byte, byte, byte, byte)) {
	for i := uint(0); i < 4; i++ {
		// Read one column at a time
		var a0, a1, a2, a3 byte
		a0 = byte((state[0] >> ((3 - i) * 8)) & 0xff)
		a1 = byte((state[1] >> ((3 - i) * 8)) & 0xff)
		a2 = byte((state[2] >> ((3 - i) * 8)) & 0xff)
		a3 = byte((state[3] >> ((3 - i) * 8)) & 0xff)

		// calculate the transformed bytes
		r0, r1, r2, r3 := calc(a0, a1, a2, a3)

		// use this mask to clear the column of its existing value
		var mask uint32
		mask = 0xff << ((3 - i) * 8)
		mask = ^mask

		// set the column with the calculated values
		state[0] = (state[0] & mask) | (uint32(r0) << ((3 - i) * 8))
		state[1] = (state[1] & mask) | (uint32(r1) << ((3 - i) * 8))
		state[2] = (state[2] & mask) | (uint32(r2) << ((3 - i) * 8))
		state[3] = (state[3] & mask) | (uint32(r3) << ((3 - i) * 8))
	}
}

// KeyExpansion is based on https://en.wikipedia.org/wiki/Rijndael_key_schedule
// I've tried to optimise for readability. For now it only supports expansion for 128-bit keys
// nwords - number of words. Values are 4, 6, 8 for 128, 192 and 256-bit
// rounds - number of rounds. Values are 10, 12, 14 for 128, 192 and 256-bit
// each round requires a 4 word key. So we need 4(10+1), 4(12+1) and 4(14+1) words in the expanded key
func keyExpansion(key []byte) []uint32 {
	nwords := 4
	rounds := 10

	expkeys := make([]uint32, 4*(rounds+1))
	// the key occupies the first nwords slots of the expanded key
	var i int
	for i < nwords {
		expkeys[i] = uint32(key[i*4])<<24 | uint32(key[i*4+1])<<16 | uint32(key[i*4+2])<<8 | uint32(key[i*4+3])
		i++
	}

	for i < 4*(rounds+1) {
		expkeys[i] = expkeys[i-1]
		expkeys[i] = rotWordLeft(expkeys[i], 1)
		expkeys[i] = subWord(expkeys[i])
		expkeys[i] ^= rcon(i/nwords - 1)
		expkeys[i] ^= expkeys[i-nwords]

		for j := 1; j <= 3; j++ {
			expkeys[i+j] = expkeys[i+j-1] ^ expkeys[i+j-nwords]
		}

		i += nwords
	}
	for j := 0; j < len(expkeys); j += 4 {
		transpose(expkeys[j : j+4])
	}

	return expkeys
}

func rcon(i int) uint32 {
	return uint32(powx[i]) << 24
}

// rotWordLeft rotates the word n bytes to the left.
func rotWordLeft(input uint32, n uint) uint32 {
	return input>>(32-8*n) | input<<(8*n)
}

// rotWordRight rotates the word n bytes to the right.
func rotWordRight(input uint32, n uint) uint32 {
	return input<<(32-8*n) | input>>(8*n)
}

func subWord(input uint32) uint32 {
	return uint32(sbox0[input>>24])<<24 |
		uint32(sbox0[input>>16&0xff])<<16 |
		uint32(sbox0[input>>8&0xff])<<8 |
		uint32(sbox0[input&0xff])
}

func invSubWord(input uint32) uint32 {
	return uint32(sbox1[input>>24])<<24 |
		uint32(sbox1[input>>16&0xff])<<16 |
		uint32(sbox1[input>>8&0xff])<<8 |
		uint32(sbox1[input&0xff])
}

func transpose(input []uint32) {
	var c0, c1, c2, c3 uint32
	for i := uint(0); i < 4; i++ {
		c0 |= (input[i] >> 24) << (8 * (3 - i))
		c1 |= (input[i] >> 16 & 0xff) << (8 * (3 - i))
		c2 |= (input[i] >> 8 & 0xff) << (8 * (3 - i))
		c3 |= (input[i] & 0xff) << (8 * (3 - i))
	}
	input[0] = c0
	input[1] = c1
	input[2] = c2
	input[3] = c3
}
