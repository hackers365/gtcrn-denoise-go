# GTCRN denoise Go

This project contains a small integration package in `./denoise`.
It now wraps `github.com/hackers365/sherpa-onnx-go/sherpa_onnx` and keeps the
existing `denoise.Config` / `Engine` / `Stream` API as a compatibility layer.

## Install

```bash
go get github.com/hackers365/gtcrn-denoise-go
```

Build and runtime native dependencies are provided by the corresponding
`hackers365/sherpa-onnx-go-{linux,macos,windows}` carrier modules. This repo no
longer vendors the modified `sherpa-onnx` native binaries itself.

## Usage

```go
import "github.com/hackers365/gtcrn-denoise-go/denoise"

engine, err := denoise.NewEngine(denoise.Config{
    ModelPath:  "./gtcrn_simple.onnx",
    PoolSize:   2,
    NumThreads: 1,
})
if err != nil {
    panic(err)
}
defer engine.Close()

stream, err := engine.NewStream()
if err != nil {
    panic(err)
}
defer stream.Close()

enhanced, err := stream.Process(pcmFloat32, 16000)
tail, err := stream.Flush()
```

Use one `Engine` per model and one `Stream` per audio route or connection.
Different streams can be processed concurrently. Calls on the same stream are
serialized by the wrapper.

The package accepts arbitrary chunk sizes. For 16 kHz audio, 20 ms is 320
samples and 60 ms is 960 samples. The native GTCRN model still runs internally
on its own frame shift, usually 256 samples.

If business audio is `int16` PCM, use:

```go
enhanced, err := stream.ProcessInt16(pcmInt16, 16000)
tail, err := stream.FlushInt16()
```

To let the caller control output buffer reuse, use the append-style APIs:

```go
buf := make([]float32, 0, engine.FrameShiftInSamples())

buf = buf[:0]
buf, err = stream.ProcessAppend(buf, pcmFloat32, 16000)

buf, err = stream.FlushAppend(buf[:0])
```

For `int16` PCM:

```go
pcmOut := make([]int16, 0, len(pcmInt16))
pcmOut, err = stream.ProcessInt16Append(pcmOut[:0], pcmInt16, 16000)
```

The native denoiser still returns a temporary C buffer that must be copied
before it is released, but the Go-side output allocation can now be owned and
reused by the caller.

For new code, prefer importing `github.com/hackers365/sherpa-onnx-go/sherpa_onnx`
directly. This module remains useful when downstream callers want to keep the
older `gtcrn-denoise-go/denoise` API surface unchanged during migration.
