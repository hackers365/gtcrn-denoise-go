package main

import (
	"log"

	"github.com/hackers365/gtcrn-denoise-go/denoise"
)

func appendSamples(dst []float32, src []float32) []float32 {
	return append(dst, src...)
}

func main() {
	log.SetFlags(log.LstdFlags | log.Lmicroseconds)

	engine, err := denoise.NewEngine(denoise.Config{
		ModelPath:  "./gtcrn_simple.onnx",
		PoolSize:   2,
		NumThreads: 1,
		Debug:      true,
	})
	if err != nil {
		log.Fatal(err)
	}
	defer engine.Close()

	const numStreams = 2
	streams := make([]*denoise.Stream, numStreams)
	outputs := make([][]float32, numStreams)
	for i := range streams {
		stream, err := engine.NewStream()
		if err != nil {
			log.Fatalf("Failed to create stream %d: %v", i, err)
		}
		streams[i] = stream
		defer streams[i].Close()
	}

	waveFilename := "./inp_16k.wav"
	wave, err := ReadWave(waveFilename)
	if err != nil {
		log.Fatalf("Failed to read %v: %v\n", waveFilename, err)
	}

	frameShift := engine.FrameShiftInSamples()
	for start := 0; start < len(wave.Samples); start += frameShift {
		end := start + frameShift
		if end > len(wave.Samples) {
			end = len(wave.Samples)
		}

		for i, stream := range streams {
			audio, err := stream.Process(wave.Samples[start:end], wave.SampleRate)
			if err != nil {
				log.Fatalf("Failed to process stream %d: %v", i, err)
			}
			outputs[i] = appendSamples(outputs[i], audio)
		}
	}

	for i, stream := range streams {
		audio, err := stream.Flush()
		if err != nil {
			log.Fatalf("Failed to flush stream %d: %v", i, err)
		}
		outputs[i] = appendSamples(outputs[i], audio)
	}

	filename := "./enhanced-online-gtcrn-shared-engine.wav"
	if err := WriteWave(filename, outputs[0], engine.SampleRate()); err != nil {
		log.Fatalf("Failed to write %v\n", filename)
	}

	log.Printf("Processed %d streams with one shared engine; saved stream 0 to %v\n",
		numStreams, filename)
}
