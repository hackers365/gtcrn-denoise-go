package denoise

import (
	"errors"
	"reflect"
	"testing"

	sherpa "github.com/hackers365/sherpa-onnx-go/sherpa_onnx"
)

func TestInt16ToFloat32AppendUsesDestinationBuffer(t *testing.T) {
	dst := make([]float32, 1, 4)
	dst[0] = 0.25

	out := Int16ToFloat32Append(dst, []int16{0, 16384})

	if len(out) != 3 {
		t.Fatalf("len(out) = %d, want 3", len(out))
	}
	if cap(out) != cap(dst) {
		t.Fatalf("cap(out) = %d, want original cap %d", cap(out), cap(dst))
	}
	want := []float32{0.25, 0, 0.5}
	if !reflect.DeepEqual(out, want) {
		t.Fatalf("out = %#v, want %#v", out, want)
	}
}

func TestFloat32ToInt16AppendUsesDestinationBuffer(t *testing.T) {
	dst := make([]int16, 1, 5)
	dst[0] = 7

	out := Float32ToInt16Append(dst, []float32{0, 0.5, 1.5, -1.5})

	if len(out) != 5 {
		t.Fatalf("len(out) = %d, want 5", len(out))
	}
	if cap(out) != cap(dst) {
		t.Fatalf("cap(out) = %d, want original cap %d", cap(out), cap(dst))
	}
	want := []int16{7, 0, 16384, 32767, -32768}
	if !reflect.DeepEqual(out, want) {
		t.Fatalf("out = %#v, want %#v", out, want)
	}
}

func TestStreamAppendMethodsReturnErrClosed(t *testing.T) {
	var stream Stream
	floatDst := []float32{0.25}
	intDst := []int16{7}

	if out, err := stream.ProcessAppend(floatDst, []float32{0}, 16000); !errors.Is(err, ErrClosed) {
		t.Fatalf("ProcessAppend err = %v, want ErrClosed", err)
	} else if !reflect.DeepEqual(out, floatDst) {
		t.Fatalf("ProcessAppend out = %#v, want original dst %#v", out, floatDst)
	}
	if out, err := stream.FlushAppend(floatDst); !errors.Is(err, ErrClosed) {
		t.Fatalf("FlushAppend err = %v, want ErrClosed", err)
	} else if !reflect.DeepEqual(out, floatDst) {
		t.Fatalf("FlushAppend out = %#v, want original dst %#v", out, floatDst)
	}
	if out, err := stream.ProcessInt16Append(intDst, []int16{0}, 16000); !errors.Is(err, ErrClosed) {
		t.Fatalf("ProcessInt16Append err = %v, want ErrClosed", err)
	} else if !reflect.DeepEqual(out, intDst) {
		t.Fatalf("ProcessInt16Append out = %#v, want original dst %#v", out, intDst)
	}
	if out, err := stream.FlushInt16Append(intDst); !errors.Is(err, ErrClosed) {
		t.Fatalf("FlushInt16Append err = %v, want ErrClosed", err)
	} else if !reflect.DeepEqual(out, intDst) {
		t.Fatalf("FlushInt16Append out = %#v, want original dst %#v", out, intDst)
	}
}

func TestToSherpaConfig(t *testing.T) {
	got := toSherpaConfig(Config{
		ModelPath:  "model.onnx",
		PoolSize:   3,
		NumThreads: 2,
		Debug:      true,
		Provider:   "cpu",
	})

	want := sherpa.OnlineSpeechDenoiserConfig{
		Model: sherpa.OnlineSpeechDenoiserModelConfig{
			Gtcrn: sherpa.OnlineSpeechDenoiserGtcrnModelConfig{
				Model: "model.onnx",
			},
			NumThreads: 2,
			Debug:      1,
			Provider:   "cpu",
		},
		PoolSize: 3,
	}

	if !reflect.DeepEqual(got, want) {
		t.Fatalf("toSherpaConfig() = %#v, want %#v", got, want)
	}
}
