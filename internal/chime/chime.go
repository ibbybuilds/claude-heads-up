// Package chime synthesizes the default alert sound. The OS notification
// sound is built to be ignorable, which is the opposite of what someone
// scrolling a feed needs. Generating it in code means no audio asset to
// license, and it works the same on every OS.
package chime

import (
	"bytes"
	"encoding/binary"
	"math"
	"os"
	"path/filepath"
)

const (
	sampleRate = 44100
	fileName   = "chime.wav"
)

// Two fast rising arpeggios (C6 E6 G6 C7). Rising pitch and repetition are
// what make an alert hard to tune out.
var (
	notes     = []float64{1046.50, 1318.51, 1567.98, 2093.00}
	noteLen   = 0.085
	burstGap  = 0.14
	bursts    = 2
	amplitude = 0.85
)

// WAV returns the chime as a 16-bit mono PCM WAV file.
func WAV() []byte {
	var samples []int16
	for b := range bursts {
		for _, freq := range notes {
			samples = append(samples, tone(freq, noteLen)...)
		}
		if b < bursts-1 {
			samples = append(samples, make([]int16, int(burstGap*sampleRate))...)
		}
	}
	return encode(samples)
}

// EnsureFile writes the chime to dir once and returns its path, because the
// OS sound APIs play files, not byte slices.
func EnsureFile(dir string) (string, error) {
	path := filepath.Join(dir, fileName)
	data := WAV()
	if existing, err := os.ReadFile(path); err == nil && bytes.Equal(existing, data) {
		return path, nil
	}
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return "", err
	}
	return path, os.WriteFile(path, data, 0o644)
}

// tone is a sine with a soft second harmonic for edge, a 3 ms attack to avoid
// clicks, and an exponential decay so each note rings like a bell.
func tone(freq, seconds float64) []int16 {
	n := int(seconds * sampleRate)
	out := make([]int16, n)
	attack := 0.003 * sampleRate
	for i := range n {
		t := float64(i) / sampleRate
		env := math.Exp(-t * 18)
		if float64(i) < attack {
			env *= float64(i) / attack
		}
		v := math.Sin(2*math.Pi*freq*t) + 0.3*math.Sin(4*math.Pi*freq*t)
		out[i] = int16(v / 1.3 * env * amplitude * math.MaxInt16)
	}
	return out
}

type wavHeader struct {
	Riff          [4]byte
	RiffSize      uint32
	Wave          [4]byte
	Fmt           [4]byte
	FmtSize       uint32
	AudioFormat   uint16
	Channels      uint16
	SampleRate    uint32
	ByteRate      uint32
	BlockAlign    uint16
	BitsPerSample uint16
	Data          [4]byte
	DataSize      uint32
}

func encode(samples []int16) []byte {
	dataSize := uint32(len(samples) * 2)
	h := wavHeader{
		Riff:          [4]byte{'R', 'I', 'F', 'F'},
		RiffSize:      36 + dataSize,
		Wave:          [4]byte{'W', 'A', 'V', 'E'},
		Fmt:           [4]byte{'f', 'm', 't', ' '},
		FmtSize:       16,
		AudioFormat:   1, // PCM
		Channels:      1,
		SampleRate:    sampleRate,
		ByteRate:      sampleRate * 2,
		BlockAlign:    2,
		BitsPerSample: 16,
		Data:          [4]byte{'d', 'a', 't', 'a'},
		DataSize:      dataSize,
	}
	var buf bytes.Buffer
	// Writes to a bytes.Buffer cannot fail.
	_ = binary.Write(&buf, binary.LittleEndian, h)
	_ = binary.Write(&buf, binary.LittleEndian, samples)
	return buf.Bytes()
}
