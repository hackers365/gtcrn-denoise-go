package denoise

/*
#cgo CFLAGS: -I${SRCDIR}/native/include
#cgo linux,amd64 LDFLAGS: -L${SRCDIR}/native/lib/linux_amd64 -lsherpa-onnx-c-api -lsherpa-onnx-core -lkaldi-decoder-core -lkaldi-native-fbank-core -lsherpa-onnx-kaldifst-core -lsherpa-onnx-fstfar -lsherpa-onnx-fst -lcargs -lkissfft-float -lssentencepiece_core -lespeak-ng -lpiper_phonemize -lucd -lonnxruntime -lstdc++ -lm -ldl
#cgo windows,amd64 LDFLAGS: -L${SRCDIR}/native/lib/windows_amd64 -lsherpa-onnx-c-api -lonnxruntime
#include "c-api.h"
#include <stdlib.h>
*/
import "C"
import "unsafe"

func createEngine(config Config) (unsafe.Pointer, int, int) {
	var c C.SherpaOnnxOnlineSpeechDenoiserConfig

	cModel := C.CString(config.ModelPath)
	defer C.free(unsafe.Pointer(cModel))
	c.model.gtcrn.model = cModel
	c.model.num_threads = C.int32_t(config.NumThreads)
	if config.Debug {
		c.model.debug = 1
	}

	var cProvider *C.char
	if config.Provider != "" {
		cProvider = C.CString(config.Provider)
		defer C.free(unsafe.Pointer(cProvider))
		c.model.provider = cProvider
	}

	engine := C.SherpaOnnxCreateOnlineSpeechDenoiserEngine(&c, C.int32_t(config.PoolSize))
	if engine == nil {
		return nil, 0, 0
	}

	sampleRate := int(C.SherpaOnnxOnlineSpeechDenoiserEngineGetSampleRate(engine))
	frameShift := int(C.SherpaOnnxOnlineSpeechDenoiserEngineGetFrameShiftInSamples(engine))

	return unsafe.Pointer(engine), sampleRate, frameShift
}

func destroyEngine(ptr unsafe.Pointer) {
	if ptr == nil {
		return
	}
	C.SherpaOnnxDestroyOnlineSpeechDenoiserEngine((*C.SherpaOnnxOnlineSpeechDenoiserEngine)(ptr))
}

func createStream(engine unsafe.Pointer) unsafe.Pointer {
	if engine == nil {
		return nil
	}
	stream := C.SherpaOnnxOnlineSpeechDenoiserEngineCreateStream(
		(*C.SherpaOnnxOnlineSpeechDenoiserEngine)(engine),
	)
	return unsafe.Pointer(stream)
}

func destroyStream(ptr unsafe.Pointer) {
	if ptr == nil {
		return
	}
	C.SherpaOnnxDestroyOnlineSpeechDenoiserStream((*C.SherpaOnnxOnlineSpeechDenoiserStream)(ptr))
}

func runStream(ptr unsafe.Pointer, samples []float32, sampleRate int) []float32 {
	if ptr == nil {
		return nil
	}

	audio := C.SherpaOnnxOnlineSpeechDenoiserStreamRun(
		(*C.SherpaOnnxOnlineSpeechDenoiserStream)(ptr),
		floatPointer(samples),
		C.int32_t(len(samples)),
		C.int32_t(sampleRate),
	)
	return denoisedAudio(audio)
}

func flushStream(ptr unsafe.Pointer) []float32 {
	if ptr == nil {
		return nil
	}

	audio := C.SherpaOnnxOnlineSpeechDenoiserStreamFlush(
		(*C.SherpaOnnxOnlineSpeechDenoiserStream)(ptr),
	)
	return denoisedAudio(audio)
}

func resetStream(ptr unsafe.Pointer) {
	if ptr == nil {
		return
	}
	C.SherpaOnnxOnlineSpeechDenoiserStreamReset((*C.SherpaOnnxOnlineSpeechDenoiserStream)(ptr))
}

func floatPointer(samples []float32) *C.float {
	if len(samples) == 0 {
		return nil
	}
	return (*C.float)(unsafe.Pointer(&samples[0]))
}

func denoisedAudio(audio *C.SherpaOnnxDenoisedAudio) []float32 {
	if audio == nil {
		return nil
	}
	defer C.SherpaOnnxDestroyDenoisedAudio(audio)

	n := int(audio.n)
	if n == 0 || audio.samples == nil {
		return nil
	}

	out := make([]float32, n)
	samples := unsafe.Slice((*float32)(unsafe.Pointer(audio.samples)), n)
	copy(out, samples)
	return out
}
