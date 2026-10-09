package chime

import (
	"bytes"
	"encoding/binary"
	"os"
	"testing"
)

func TestWAVHeader(t *testing.T) {
	wav := WAV()
	if !bytes.HasPrefix(wav, []byte("RIFF")) || string(wav[8:16]) != "WAVEfmt " {
		t.Fatal("missing RIFF/WAVE header")
	}
	riffSize := binary.LittleEndian.Uint32(wav[4:8])
	if int(riffSize) != len(wav)-8 {
		t.Errorf("RIFF size %d, file is %d bytes", riffSize, len(wav))
	}
	dataSize := binary.LittleEndian.Uint32(wav[40:44])
	if int(dataSize) != len(wav)-44 {
		t.Errorf("data size %d, payload is %d bytes", dataSize, len(wav)-44)
	}
}

func TestWAVIsLoudButNotClipped(t *testing.T) {
	wav := WAV()
	var peak int16
	for i := 44; i+1 < len(wav); i += 2 {
		s := int16(binary.LittleEndian.Uint16(wav[i:]))
		if s < 0 {
			s = -s
		}
		peak = max(peak, s)
	}
	if peak < 20000 {
		t.Errorf("peak %d is too quiet to grab attention", peak)
	}
}

func TestEnsureFileIsIdempotent(t *testing.T) {
	dir := t.TempDir()
	path, err := EnsureFile(dir)
	if err != nil {
		t.Fatal(err)
	}
	first, _ := os.Stat(path)
	if _, err := EnsureFile(dir); err != nil {
		t.Fatal(err)
	}
	second, _ := os.Stat(path)
	if !first.ModTime().Equal(second.ModTime()) {
		t.Error("unchanged chime was rewritten")
	}
}
