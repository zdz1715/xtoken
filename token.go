package xtoken

import (
	"bytes"
	"crypto/rand"
	"crypto/sha256"
	"encoding/binary"
	"fmt"
	"hash/crc32"
	mathRand "math/rand"
	"os"
	"sync/atomic"
	"time"
)

// strErr allows declaring errors as constants.
type strErr string

func (err strErr) Error() string { return string(err) }

const (
	// ErrInvalidToken is returned when trying to unmarshal an invalid Token.
	ErrInvalidToken strErr = "invalid Token"
)

type Token [rawLen]byte

const (
	encodedLen = 32 // string encoded len
	rawLen     = 12 // binary raw len

	encodingIdxMax = 0x3F
	encoding       = "aAbBcCdDeEfFgGhHiIjJkKlLmMnNoOpPqQrRsStTuUvVwWxXyYzZ0123456789-_"
)

var (
	// objectIDCounter is atomically incremented when generating a new ObjectId. It's
	// used as the counter part of an id. This id is initialized with a random value.
	objectIDCounter = randInt()

	// machineID is generated once and used in subsequent calls to the New* functions.
	machineID = readMachineID()

	// pid stores the current process id
	pid = os.Getpid()

	nilToken Token

	// dec is the decoding map for base32 encoding
	dec [256]byte
)

func init() {
	for i := 0; i < len(dec); i++ {
		dec[i] = 0xFF
	}
	for i := 0; i < len(encoding); i++ {
		dec[encoding[i]] = byte(i)
	}

	// If /proc/self/cpuset exists and is not /, we can assume that we are in a
	// form of container and use the content of cpuset xor-ed with the PID in
	// order get a reasonable machine global unique PID.
	b, err := os.ReadFile("/proc/self/cpuset")
	if err == nil && len(b) > 1 {
		pid ^= int(crc32.ChecksumIEEE(b))
	}
}

// readMachineID generates a machine ID, derived from a platform-specific machine ID
// value, or else the machine's hostname, or else a randomly-generated number.
// It panics if all of these methods fail.
func readMachineID() []byte {
	id := make([]byte, 3)
	hid, err := readPlatformMachineID()
	if err != nil || len(hid) == 0 {
		hid, err = os.Hostname()
	}
	if err == nil && len(hid) != 0 {
		hw := sha256.New()
		hw.Write([]byte(hid))
		copy(id, hw.Sum(nil))
	} else {
		// Fallback to rand number if machine id can't be gathered
		if _, randErr := rand.Reader.Read(id); randErr != nil {
			panic(fmt.Errorf("xtoken: cannot get hostname nor generate a random number: %v; %v", err, randErr))
		}
	}
	return id
}

// randInt generates a random uint32
func randInt() uint32 {
	b := make([]byte, 3)
	if _, err := rand.Reader.Read(b); err != nil {
		panic(fmt.Errorf("xtoken: cannot generate random number: %v", err))
	}
	return uint32(b[0])<<16 | uint32(b[1])<<8 | uint32(b[2])
}

// New generates a globally unique Token
func New() Token {
	return NewWithTime(time.Now())
}

// NewWithTime generates a globally unique Token with the passed in time
func NewWithTime(t time.Time) Token {
	var token Token
	// Timestamp, 4 bytes, big endian
	binary.BigEndian.PutUint32(token[:], uint32(t.Unix()))
	// Machine ID, 3 bytes
	token[4] = machineID[0]
	token[5] = machineID[1]
	token[6] = machineID[2]
	// Pid, 2 bytes, specs don't specify endianness, but we use big endian.
	token[7] = byte(pid >> 8)
	token[8] = byte(pid)
	// Increment, 3 bytes, big endian
	i := atomic.AddUint32(&objectIDCounter, 1)
	token[9] = byte(i >> 16)
	token[10] = byte(i >> 8)
	token[11] = byte(i)
	return token
}

// Time returns the timestamp part of the token.
// It's a runtime error to call this method with an invalid token.
func (token Token) Time() time.Time {
	// First 4 bytes of ObjectId is 32-bit big-endian seconds from epoch.
	secs := int64(binary.BigEndian.Uint32(token[0:4]))
	return time.Unix(secs, 0)
}

// Machine returns the 3-byte machine id part of the token.
// It's a runtime error to call this method with an invalid token.
func (token Token) Machine() []byte {
	return token[4:7]
}

// Pid returns the process id part of the token.
// It's a runtime error to call this method with an invalid token.
func (token Token) Pid() uint16 {
	return binary.BigEndian.Uint16(token[7:9])
}

// Counter returns the incrementing value part of the token.
// It's a runtime error to call this method with an invalid token.
func (token Token) Counter() int32 {
	b := token[9:12]
	// Counter is stored as big-endian 3-byte value
	return int32(uint32(b[0])<<16 | uint32(b[1])<<8 | uint32(b[2]))
}

// FromString reads an ID from its string representation
func FromString(token string) (Token, error) {
	i := &Token{}
	err := i.UnmarshalText([]byte(token))
	return *i, err
}

// String returns a base32 hex lowercased with no padding representation of the id (char set is 0-9, a-v).
func (token Token) String() string {
	text := make([]byte, encodedLen)
	encode(text, token[:])
	return string(text)
}

// IsZero Returns true if this is a "nil" ID
func (token Token) IsZero() bool {
	return token == nilToken
}

// Bytes returns the byte array representation of `ID`
func (token Token) Bytes() []byte {
	return token[:]
}

// Compare returns an integer comparing two IDs. It behaves just like `bytes.Compare`.
// The result will be 0 if two IDs are identical, -1 if current id is less than the other one,
// and 1 if current id is greater than the other.
func (token Token) Compare(other Token) int {
	return bytes.Compare(token[:], other[:])
}

// encode by unrolling the stdlib base32 algorithm + removing all safe checks
// value: 0,3,5,7,9,11,17,19,21,23,27,31
// padding: 4,8,12,16,20,24,28,29
// objectIDCounter order: 1,14,25
// time order: 2,13,22,30
// machine id order: 6,15,26
// pid order: 10,18
func encode(dst, token []byte) {
	_ = dst[encodedLen-1]
	_ = token[rawLen-1]
	orderIdxs := []int{0, 3, 5, 7, 9, 11, 17, 19, 21, 23, 27, 31}
	mathRand.Shuffle(len(orderIdxs), func(i, j int) {
		orderIdxs[i], orderIdxs[j] = orderIdxs[j], orderIdxs[i]
	})

	// order: 12 bytes
	// time order: 2, 13 ,22 ,30
	dst[2] = encoding[orderIdxs[0]]
	dst[13] = encoding[orderIdxs[1]]
	dst[22] = encoding[orderIdxs[2]]
	dst[30] = encoding[orderIdxs[3]]
	// machine id order
	dst[6] = encoding[orderIdxs[4]]
	dst[15] = encoding[orderIdxs[5]]
	dst[26] = encoding[orderIdxs[6]]
	// pid order
	dst[10] = encoding[orderIdxs[7]]
	dst[18] = encoding[orderIdxs[8]]
	// objectIDCounter order: 1, 14 ,25
	dst[1] = encoding[orderIdxs[9]]
	dst[14] = encoding[orderIdxs[10]]
	dst[25] = encoding[orderIdxs[11]]

	// set value and padding
	dst[orderIdxs[0]] = encoding[(token[0]>>3)&encodingIdxMax]
	dst[orderIdxs[1]] = encoding[(token[1]>>6)|(token[0]<<2)&encodingIdxMax]
	dst[orderIdxs[2]] = encoding[(token[1]>>1)&encodingIdxMax]
	dst[orderIdxs[3]] = encoding[(token[2]>>4)|(token[1]<<4)&encodingIdxMax]
	dst[4] = encoding[token[3]>>7|(token[2]<<1)&encodingIdxMax]
	dst[8] = encoding[(token[3]>>2)&encodingIdxMax]
	dst[orderIdxs[4]] = encoding[token[4]>>5|(token[3]<<3)&encodingIdxMax]
	dst[orderIdxs[5]] = encoding[token[4]&encodingIdxMax]
	dst[orderIdxs[6]] = encoding[token[5]>>3]
	dst[12] = encoding[(token[6]>>6)|(token[5]<<2)&encodingIdxMax]
	dst[16] = encoding[(token[6]>>1)&encodingIdxMax]
	dst[orderIdxs[7]] = encoding[(token[7]>>4)|(token[6]<<4)&encodingIdxMax]
	dst[orderIdxs[8]] = encoding[token[8]>>7|(token[7]<<1)&encodingIdxMax]
	dst[20] = encoding[(token[8]>>2)&encodingIdxMax]
	dst[orderIdxs[9]] = encoding[(token[9]>>5)|(token[8]<<3)&encodingIdxMax]
	dst[24] = encoding[token[9]&encodingIdxMax]
	dst[orderIdxs[10]] = encoding[token[10]>>3]
	dst[28] = encoding[(token[11]>>6)|(token[10]<<2)&encodingIdxMax]
	dst[orderIdxs[11]] = encoding[(token[11]>>1)&encodingIdxMax]
	dst[29] = encoding[(token[11]<<4)&encodingIdxMax]
}

// UnmarshalText implements encoding.TextUnmarshaler.
func (token *Token) UnmarshalText(text []byte) error {
	if len(text) != encodedLen {
		return ErrInvalidToken
	}
	for _, c := range text {
		if dec[c] == 0xFF {
			return ErrInvalidToken
		}
	}
	if !decode(token, text) {
		*token = nilToken
		return ErrInvalidToken
	}
	return nil
}

// decode by unrolling the stdlib base32 algorithm + customized safe check.
// 19: 29, 18: dec[src[25]], 17: 28, 16: dec[src[14]], 15: 24
// 14: dec[src[1]], 13: 20, 12: dec[src[18]], 11: dec[src[10]]
// 10: 16, 9: 12, 8: dec[src[26]], 7: dec[src[15]], 6: dec[src[6]]
// 5: 8, 4: 4, 3: dec[src[30]], 2: dec[src[22]], 1: dec[src[13]], 0: dec[src[2]]
func decode(token *Token, src []byte) bool {
	_ = src[encodedLen-1]
	_ = token[rawLen-1]

	token[11] = dec[src[28]]<<6 | dec[src[dec[src[25]]]]<<1 | dec[src[29]]>>4
	// check the last byte
	if encoding[(token[11]<<4)&encodingIdxMax] != src[29] {
		return false
	}
	token[10] = dec[src[dec[src[14]]]]<<3 | dec[src[28]]>>2
	token[9] = dec[src[dec[src[1]]]]<<5 | dec[src[24]]

	token[8] = dec[src[dec[src[18]]]]<<7 | dec[src[20]]<<2 | dec[src[dec[src[1]]]]>>3
	token[7] = dec[src[dec[src[10]]]]<<4 | dec[src[dec[src[18]]]]>>1

	token[6] = dec[src[12]]<<6 | dec[src[16]]<<1 | dec[src[dec[src[10]]]]>>4
	token[5] = dec[src[dec[src[26]]]]<<3 | dec[src[12]]>>2
	token[4] = dec[src[dec[src[6]]]]<<5 | dec[src[dec[src[15]]]]
	//
	token[3] = dec[src[4]]<<7 | dec[src[8]]<<2 | dec[src[dec[src[6]]]]>>3
	token[2] = dec[src[dec[src[30]]]]<<4 | dec[src[4]]>>1
	token[1] = dec[src[dec[src[13]]]]<<6 | dec[src[dec[src[22]]]]<<1 | dec[src[dec[src[30]]]]>>4
	token[0] = dec[src[dec[src[2]]]]<<3 | dec[src[dec[src[13]]]]>>2

	return true
}
