/*The MIT-Zero License

Copyright (c) 2014 Sarun Rattanasiri

Permission is hereby granted, free of charge, to any person obtaining a copy
of this software and associated documentation files (the "Software"), to deal
in the Software without restriction, including without limitation the rights
to use, copy, modify, merge, publish, distribute, sublicense, and/or sell
copies of the Software, and to permit persons to whom the Software is
furnished to do so.

THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE
AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM,
OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN
THE SOFTWARE.*/

package ige

import "crypto/cipher"

type ige struct {
	b              cipher.Block
	blockSize      int
	lastplaintext  []byte
	lastciphertext []byte
}

//test vector @ https://web.archive.org/web/20120418022623/http://www.links.org/files/openssl-ige.pdf

func newIGE(b cipher.Block, iv []byte) (ret *ige) {
	//in IGE mode len(iv) is 2 * block size
	blockSize := b.BlockSize()
	internal := make([]byte, len(iv))
	copy(internal, iv)
	ret = &ige{
		b:              b,
		blockSize:      blockSize,
		lastciphertext: internal[:blockSize],
		lastplaintext:  internal[blockSize:],
	}
	return
}

type igeEncrypter ige

//cryptBlocks was written in Encrypter perspective
//since IGE mode use nearly identical algorithm
//we can use it as Decrypter too
func cryptBlocks(dst, src []byte, x *igeEncrypter, cryptFunc func([]byte, []byte)) {
	if len(src)%x.blockSize != 0 {
		panic("crypto/cipher: input not full blocks")
	}
	if len(dst) < len(src) {
		panic("crypto/cipher: output smaller than input")
	}

	for len(src) > 0 {
		//use lastciphertext as scratch memory
		for i := 0; i < x.blockSize; i++ {
			x.lastciphertext[i] ^= src[i]
		}
		cryptFunc(x.lastciphertext, x.lastciphertext) //x.b.Encrypt
		for i := 0; i < x.blockSize; i++ {
			x.lastciphertext[i] ^= x.lastplaintext[i]
		}

		//update internal state
		copy(x.lastplaintext, src)
		//copy to destination
		copy(dst, x.lastciphertext)

		src = src[x.blockSize:]
		dst = dst[x.blockSize:]
	}
}

// NewIGEEncrypter returns a BlockMode which encrypts in
// infinite garble extension mode, using the given Block.
// The length of iv must be 2 times of Block's block size.
func NewIGEEncrypter(b cipher.Block, iv []byte) cipher.BlockMode {
	if len(iv) != b.BlockSize()*2 {
		panic("ige.NewIGEEncrypter: IV length must equal 2 * block size")
	}
	return (*igeEncrypter)(newIGE(b, iv))
}

func (x *igeEncrypter) BlockSize() int { return x.blockSize }

func (x *igeEncrypter) CryptBlocks(dst, src []byte) {
	//just alias
	cryptBlocks(dst, src, x, x.b.Encrypt)
}

func (x *igeEncrypter) SetIV(iv []byte) {
	if len(iv) != x.blockSize*2 {
		panic("cipher: incorrect length IV")
	}
	copy(x.lastplaintext, iv[:x.blockSize])
	copy(x.lastciphertext, iv[x.blockSize:])
}

type igeDecrypter ige

// NewIGEDecrypter returns a BlockMode which decrypts in
// infinite garble extension mode, using the given Block.
// The length of iv must be 2 times of Block's block size.
func NewIGEDecrypter(b cipher.Block, iv []byte) cipher.BlockMode {
	if len(iv) != b.BlockSize()*2 {
		panic("cipher.NewIGEDecrypter: IV length must equal 2 * block size")
	}
	return (*igeDecrypter)(newIGE(b, iv))
}

func (x *igeDecrypter) BlockSize() int { return x.blockSize }

func (x *igeDecrypter) CryptBlocks(dst, src []byte) {
	//swap block to make final x.lastciphertext, x.lastplaintext make sense
	var tmp []byte
	tmp = x.lastciphertext
	x.lastciphertext = x.lastplaintext
	x.lastplaintext = tmp
	cryptBlocks(dst, src, (*igeEncrypter)(x), x.b.Decrypt)
	x.lastplaintext = x.lastciphertext
	x.lastciphertext = tmp
	tmp = nil
}

func (x *igeDecrypter) SetIV(iv []byte) {
	((*igeEncrypter)(x)).SetIV(iv)
}
