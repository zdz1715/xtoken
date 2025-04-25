package xtoken

import (
	"bytes"
	"fmt"
	"testing"
	"time"
)

func TestNewWithTime(t *testing.T) {

	token := NewWithTime(time.Now())

	t.Logf("Token: %s, Time: %s, Machine: %d, Pid: %d, Counter: %d", token, token.Time(), token.Machine(), token.Pid(), token.Counter())

	tokenFormat, err := FromString(token.String())
	if err != nil {
		t.Fatalf("err: %v", err)
	}
	t.Logf("Time: %s, Machine: %d, Pid: %d, Counter: %d",
		tokenFormat.Time(), tokenFormat.Machine(), tokenFormat.Pid(), tokenFormat.Counter())
}

type IDParts struct {
	token     Token
	timestamp int64
	machine   []byte
	pid       uint16
	counter   int32
}

var IDs = []IDParts{
	{
		Token{0x4d, 0x88, 0xe1, 0x5b, 0x60, 0xf4, 0x86, 0xe4, 0x28, 0x41, 0x2d, 0xc9},
		1300816219,
		[]byte{0x60, 0xf4, 0x86},
		0xe428,
		4271561,
	},
	{
		Token{0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00},
		0,
		[]byte{0x00, 0x00, 0x00},
		0x0000,
		0,
	},
	{
		Token{0x00, 0x00, 0x00, 0x00, 0xaa, 0xbb, 0xcc, 0xdd, 0xee, 0x00, 0x00, 0x01},
		0,
		[]byte{0xaa, 0xbb, 0xcc},
		0xddee,
		1,
	},
}

func TestIDPartsExtraction(t *testing.T) {
	for i, v := range IDs {
		t.Run(fmt.Sprintf("Test%d", i), func(t *testing.T) {
			if got, want := v.token.Time(), time.Unix(v.timestamp, 0); got != want {
				t.Errorf("Time() = %v, want %v", got, want)
			}
			if got, want := v.token.Machine(), v.machine; !bytes.Equal(got, want) {
				t.Errorf("Machine() = %v, want %v", got, want)
			}
			if got, want := v.token.Pid(), v.pid; got != want {
				t.Errorf("Pid() = %v, want %v", got, want)
			}
			if got, want := v.token.Counter(), v.counter; got != want {
				t.Errorf("Counter() = %v, want %v", got, want)
			}
		})
	}
}

func TestNew(t *testing.T) {
	// Generate 10 tokens
	tokens := make([]Token, 10000)
	for i := 0; i < 10000; i++ {
		tokens[i] = New()
	}
	for i := 1; i < 10000; i++ {
		prevToken := tokens[i-1]
		token := tokens[i]
		// Test for uniqueness among all other 9 generated ids
		for j, tt := range tokens {
			if j != i {
				if token.Compare(tt) == 0 {
					t.Errorf("generated Token is not unique (%d/%d)", i, j)
				}
			}
		}
		// Check that timestamp was incremented and is within 30 seconds of the previous one
		secs := token.Time().Sub(prevToken.Time()).Seconds()
		if secs < 0 || secs > 30 {
			t.Error("wrong timestamp in generated ID")
		}
		// Check that machine ids are the same
		if !bytes.Equal(token.Machine(), prevToken.Machine()) {
			t.Error("machine ID not equal")
		}
		// Check that pids are the same
		if token.Pid() != prevToken.Pid() {
			t.Error("pid not equal")
		}
		// Test for proper increment
		if got, want := int(token.Counter()-prevToken.Counter()), 1; got != want {
			t.Errorf("wrong increment in generated ID, delta=%v, want %v", got, want)
		}
	}
}

func BenchmarkNew(b *testing.B) {
	b.ReportAllocs()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			_ = New()
		}
	})
}

func BenchmarkNewString(b *testing.B) {
	b.ReportAllocs()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			_ = New().String()
		}
	})
}
