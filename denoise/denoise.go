package denoise

import (
	"errors"
	"math"
	"sync"
	"unsafe"
)

var (
	ErrClosed = errors.New("denoise: resource is closed")
)

type Config struct {
	ModelPath  string
	PoolSize   int
	NumThreads int
	Debug      bool
	Provider   string
}

type Engine struct {
	mu         sync.Mutex
	impl       unsafe.Pointer
	sampleRate int
	frameShift int
}

type Stream struct {
	mu         sync.Mutex
	impl       unsafe.Pointer
	sampleRate int
}

func NewEngine(config Config) (*Engine, error) {
	if config.ModelPath == "" {
		return nil, errors.New("denoise: model path is empty")
	}

	impl, sampleRate, frameShift := createEngine(config)
	if impl == nil {
		return nil, errors.New("denoise: failed to create online speech denoiser engine")
	}

	return &Engine{
		impl:       impl,
		sampleRate: sampleRate,
		frameShift: frameShift,
	}, nil
}

func (e *Engine) NewStream() (*Stream, error) {
	e.mu.Lock()
	defer e.mu.Unlock()

	if e.impl == nil {
		return nil, ErrClosed
	}

	impl := createStream(e.impl)
	if impl == nil {
		return nil, errors.New("denoise: failed to create stream")
	}

	return &Stream{
		impl:       impl,
		sampleRate: e.sampleRate,
	}, nil
}

func (e *Engine) SampleRate() int {
	e.mu.Lock()
	defer e.mu.Unlock()
	return e.sampleRate
}

func (e *Engine) FrameShiftInSamples() int {
	e.mu.Lock()
	defer e.mu.Unlock()
	return e.frameShift
}

func (e *Engine) Close() {
	e.mu.Lock()
	defer e.mu.Unlock()

	if e.impl == nil {
		return
	}

	destroyEngine(e.impl)
	e.impl = nil
}

func (s *Stream) Process(samples []float32, sampleRate int) ([]float32, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.impl == nil {
		return nil, ErrClosed
	}

	if sampleRate <= 0 {
		sampleRate = s.sampleRate
	}

	return runStream(s.impl, samples, sampleRate), nil
}

func (s *Stream) ProcessInt16(samples []int16, sampleRate int) ([]int16, error) {
	enhanced, err := s.Process(Int16ToFloat32(samples), sampleRate)
	if err != nil {
		return nil, err
	}

	return Float32ToInt16(enhanced), nil
}

func (s *Stream) Flush() ([]float32, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.impl == nil {
		return nil, ErrClosed
	}

	return flushStream(s.impl), nil
}

func (s *Stream) FlushInt16() ([]int16, error) {
	enhanced, err := s.Flush()
	if err != nil {
		return nil, err
	}

	return Float32ToInt16(enhanced), nil
}

func (s *Stream) Reset() error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.impl == nil {
		return ErrClosed
	}

	resetStream(s.impl)
	return nil
}

func (s *Stream) Close() {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.impl == nil {
		return
	}

	destroyStream(s.impl)
	s.impl = nil
}

func Int16ToFloat32(samples []int16) []float32 {
	if len(samples) == 0 {
		return nil
	}

	out := make([]float32, len(samples))
	for i, sample := range samples {
		out[i] = float32(sample) / 32768.0
	}
	return out
}

func Float32ToInt16(samples []float32) []int16 {
	if len(samples) == 0 {
		return nil
	}

	out := make([]int16, len(samples))
	for i, sample := range samples {
		if sample >= 1 {
			out[i] = math.MaxInt16
			continue
		}
		if sample <= -1 {
			out[i] = math.MinInt16
			continue
		}
		out[i] = int16(math.Round(float64(sample * 32767.0)))
	}
	return out
}
