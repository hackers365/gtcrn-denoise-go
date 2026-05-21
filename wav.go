package main

import (
	"encoding/binary"
	"errors"
	"fmt"
	"io"
	"math"
	"os"
)

type Wave struct {
	Samples    []float32
	SampleRate int
}

func ReadWave(filename string) (*Wave, error) {
	f, err := os.Open(filename)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	var riff [12]byte
	if _, err := io.ReadFull(f, riff[:]); err != nil {
		return nil, err
	}
	if string(riff[0:4]) != "RIFF" || string(riff[8:12]) != "WAVE" {
		return nil, errors.New("not a RIFF/WAVE file")
	}

	var audioFormat uint16
	var numChannels uint16
	var sampleRate uint32
	var bitsPerSample uint16
	var data []byte

	for {
		var header [8]byte
		if _, err := io.ReadFull(f, header[:]); err != nil {
			if errors.Is(err, io.EOF) || errors.Is(err, io.ErrUnexpectedEOF) {
				break
			}
			return nil, err
		}

		chunkID := string(header[0:4])
		chunkSize := binary.LittleEndian.Uint32(header[4:8])
		chunk := make([]byte, chunkSize)
		if _, err := io.ReadFull(f, chunk); err != nil {
			return nil, err
		}
		if chunkSize%2 == 1 {
			if _, err := f.Seek(1, io.SeekCurrent); err != nil {
				return nil, err
			}
		}

		switch chunkID {
		case "fmt ":
			if len(chunk) < 16 {
				return nil, errors.New("invalid fmt chunk")
			}
			audioFormat = binary.LittleEndian.Uint16(chunk[0:2])
			numChannels = binary.LittleEndian.Uint16(chunk[2:4])
			sampleRate = binary.LittleEndian.Uint32(chunk[4:8])
			bitsPerSample = binary.LittleEndian.Uint16(chunk[14:16])
		case "data":
			data = chunk
		}
	}

	if numChannels != 1 {
		return nil, fmt.Errorf("expected mono WAV, got %d channels", numChannels)
	}
	if sampleRate == 0 || len(data) == 0 {
		return nil, errors.New("missing WAV fmt or data chunk")
	}

	switch {
	case audioFormat == 1 && bitsPerSample == 16:
		samples := make([]float32, len(data)/2)
		for i := range samples {
			v := int16(binary.LittleEndian.Uint16(data[2*i : 2*i+2]))
			samples[i] = float32(v) / 32768.0
		}
		return &Wave{Samples: samples, SampleRate: int(sampleRate)}, nil
	case audioFormat == 3 && bitsPerSample == 32:
		samples := make([]float32, len(data)/4)
		for i := range samples {
			bits := binary.LittleEndian.Uint32(data[4*i : 4*i+4])
			samples[i] = math.Float32frombits(bits)
		}
		return &Wave{Samples: samples, SampleRate: int(sampleRate)}, nil
	default:
		return nil, fmt.Errorf("unsupported WAV format: audio_format=%d bits_per_sample=%d", audioFormat, bitsPerSample)
	}
}

func WriteWave(filename string, samples []float32, sampleRate int) error {
	f, err := os.Create(filename)
	if err != nil {
		return err
	}
	defer f.Close()

	dataBytes := uint32(len(samples) * 2)
	riffSize := uint32(36) + dataBytes

	if _, err := f.Write([]byte("RIFF")); err != nil {
		return err
	}
	if err := binary.Write(f, binary.LittleEndian, riffSize); err != nil {
		return err
	}
	if _, err := f.Write([]byte("WAVEfmt ")); err != nil {
		return err
	}

	if err := binary.Write(f, binary.LittleEndian, uint32(16)); err != nil {
		return err
	}
	if err := binary.Write(f, binary.LittleEndian, uint16(1)); err != nil {
		return err
	}
	if err := binary.Write(f, binary.LittleEndian, uint16(1)); err != nil {
		return err
	}
	if err := binary.Write(f, binary.LittleEndian, uint32(sampleRate)); err != nil {
		return err
	}
	if err := binary.Write(f, binary.LittleEndian, uint32(sampleRate*2)); err != nil {
		return err
	}
	if err := binary.Write(f, binary.LittleEndian, uint16(2)); err != nil {
		return err
	}
	if err := binary.Write(f, binary.LittleEndian, uint16(16)); err != nil {
		return err
	}
	if _, err := f.Write([]byte("data")); err != nil {
		return err
	}
	if err := binary.Write(f, binary.LittleEndian, dataBytes); err != nil {
		return err
	}

	for _, sample := range samples {
		if sample > 1 {
			sample = 1
		}
		if sample < -1 {
			sample = -1
		}
		if err := binary.Write(f, binary.LittleEndian, int16(math.Round(float64(sample*32767.0)))); err != nil {
			return err
		}
	}

	return nil
}
