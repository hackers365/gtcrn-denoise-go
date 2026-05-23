package denoise

import (
	"errors"
	"math"
	"sync"

	sherpa "github.com/hackers365/sherpa-onnx-go/sherpa_onnx"
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
	impl       *sherpa.OnlineSpeechDenoiserEngine
	sampleRate int
	frameShift int
}

type Stream struct {
	mu         sync.Mutex
	impl       *sherpa.OnlineSpeechDenoiserStream
	sampleRate int
}

func NewEngine(config Config) (*Engine, error) {
	if config.ModelPath == "" {
		return nil, errors.New("denoise: model path is empty")
	}

	sherpaConfig := toSherpaConfig(config)
	impl := sherpa.NewOnlineSpeechDenoiserEngine(&sherpaConfig)
	if impl == nil {
		return nil, errors.New("denoise: failed to create online speech denoiser engine")
	}

	return &Engine{
		impl:       impl,
		sampleRate: impl.SampleRate(),
		frameShift: impl.FrameShiftInSamples(),
	}, nil
}

func (e *Engine) NewStream() (*Stream, error) {
	e.mu.Lock()
	defer e.mu.Unlock()

	if e.impl == nil {
		return nil, ErrClosed
	}

	impl := e.impl.CreateStream()
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

	sherpa.DeleteOnlineSpeechDenoiserEngine(e.impl)
	e.impl = nil
}

func (s *Stream) Process(samples []float32, sampleRate int) ([]float32, error) {
	return s.ProcessAppend(nil, samples, sampleRate)
}

func (s *Stream) ProcessAppend(dst []float32, samples []float32, sampleRate int) ([]float32, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.impl == nil {
		return dst, ErrClosed
	}

	if sampleRate <= 0 {
		sampleRate = s.sampleRate
	}

	return appendDenoisedAudio(dst, s.impl.Run(samples, sampleRate)), nil
}

func (s *Stream) ProcessInt16(samples []int16, sampleRate int) ([]int16, error) {
	return s.ProcessInt16Append(nil, samples, sampleRate)
}

func (s *Stream) ProcessInt16Append(dst []int16, samples []int16, sampleRate int) ([]int16, error) {
	enhanced, err := s.Process(Int16ToFloat32(samples), sampleRate)
	if err != nil {
		return dst, err
	}

	return Float32ToInt16Append(dst, enhanced), nil
}

func (s *Stream) Flush() ([]float32, error) {
	return s.FlushAppend(nil)
}

func (s *Stream) FlushAppend(dst []float32) ([]float32, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.impl == nil {
		return dst, ErrClosed
	}

	return appendDenoisedAudio(dst, s.impl.Flush()), nil
}

func (s *Stream) FlushInt16() ([]int16, error) {
	return s.FlushInt16Append(nil)
}

func (s *Stream) FlushInt16Append(dst []int16) ([]int16, error) {
	enhanced, err := s.Flush()
	if err != nil {
		return dst, err
	}

	return Float32ToInt16Append(dst, enhanced), nil
}

func (s *Stream) Reset() error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.impl == nil {
		return ErrClosed
	}

	s.impl.Reset()
	return nil
}

func (s *Stream) Close() {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.impl == nil {
		return
	}

	sherpa.DeleteOnlineSpeechDenoiserStream(s.impl)
	s.impl = nil
}

func toSherpaConfig(config Config) sherpa.OnlineSpeechDenoiserConfig {
	return sherpa.OnlineSpeechDenoiserConfig{
		Model: sherpa.OnlineSpeechDenoiserModelConfig{
			Gtcrn: sherpa.OnlineSpeechDenoiserGtcrnModelConfig{
				Model: config.ModelPath,
			},
			NumThreads: int32(config.NumThreads),
			Debug:      boolToDebugInt32(config.Debug),
			Provider:   config.Provider,
		},
		PoolSize: int32(config.PoolSize),
	}
}

func boolToDebugInt32(v bool) int32 {
	if v {
		return 1
	}
	return 0
}

func appendDenoisedAudio(dst []float32, audio *sherpa.DenoisedAudio) []float32 {
	if audio == nil || len(audio.Samples) == 0 {
		return dst
	}

	base := len(dst)
	out := growFloat32(dst, len(audio.Samples))
	copy(out[base:], audio.Samples)
	return out
}

func Int16ToFloat32(samples []int16) []float32 {
	return Int16ToFloat32Append(nil, samples)
}

func Int16ToFloat32Append(dst []float32, samples []int16) []float32 {
	if len(samples) == 0 {
		return dst
	}

	base := len(dst)
	out := growFloat32(dst, len(samples))
	for i, sample := range samples {
		out[base+i] = float32(sample) / 32768.0
	}
	return out
}

func Float32ToInt16(samples []float32) []int16 {
	return Float32ToInt16Append(nil, samples)
}

func Float32ToInt16Append(dst []int16, samples []float32) []int16 {
	if len(samples) == 0 {
		return dst
	}

	base := len(dst)
	out := growInt16(dst, len(samples))
	for i, sample := range samples {
		if sample >= 1 {
			out[base+i] = math.MaxInt16
			continue
		}
		if sample <= -1 {
			out[base+i] = math.MinInt16
			continue
		}
		out[base+i] = int16(math.Round(float64(sample * 32767.0)))
	}
	return out
}

func growFloat32(dst []float32, n int) []float32 {
	need := len(dst) + n
	if need <= cap(dst) {
		return dst[:need]
	}

	out := make([]float32, need)
	copy(out, dst)
	return out
}

func growInt16(dst []int16, n int) []int16 {
	need := len(dst) + n
	if need <= cap(dst) {
		return dst[:need]
	}

	out := make([]int16, need)
	copy(out, dst)
	return out
}
