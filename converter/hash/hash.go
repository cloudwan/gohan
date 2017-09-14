// Copyright (C) 2017 NTT Innovation Institute, Inc.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//    http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or
// implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package hash

const (
	mod  uint32 = 0xFFFFFFFB
	base uint32 = 0x101
)

// Hash is a struct for calculating hashing strings
type Hash struct {
	powers []uint32
}

// Node represents hashed value and length of a string
type Node struct {
	value  uint32
	length int
}

// AddMod adds two numbers in Z mod
// assuming 0 <= x,y < mod
func AddMod(x, y, mod uint32) uint32 {
	result := uint64(x) + uint64(y)
	mod64 := uint64(mod)
	if result >= mod64 {
		result -= mod64
	}
	return uint32(result)
}

// MulMod multiplies two numbers in Z mod
func MulMod(x, y, mod uint32) uint32 {
	result := uint64(x) * uint64(y)
	mod64 := uint64(mod)
	if result >= mod64 {
		result %= mod64
	}
	return uint32(result)
}

// Calc calculates a hash of a string
func (hash *Hash) Calc(string string) Node {
	length := len(string)
	hash.getPowers(length)
	var value uint32
	for i, char := range string {
		value = add(value, mul(uint32(char), hash.powers[i]))
	}
	return Node{value, length}
}

// Join calculates hash of two joined strings
func (hash *Hash) Join(first, second Node) Node {
	hash.getPowers(first.length + 1)
	return Node{
		add(first.value, mul(second.value, hash.powers[first.length])),
		first.length + second.length,
	}
}

func (hash *Hash) getPowers(limit int) {
	length := len(hash.powers)
	if length < 1 {
		hash.powers = []uint32{1}
		length = 1
	}
	for i := length; i < limit; i++ {
		hash.powers = append(hash.powers, mul(hash.powers[i-1], base))
	}
}

func add(x, y uint32) uint32 {
	return AddMod(x, y, mod)
}

func mul(x, y uint32) uint32 {
	return MulMod(x, y, mod)
}
