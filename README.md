# GTCRN denoise Go

This project contains a small integration package in `./denoise`.
It calls the modified sherpa-onnx C API directly through cgo, so it does not
depend on `github.com/k2-fsa/sherpa-onnx-go` or the local `scripts/go/_internal`
wrapper.

## Install

```bash
go get github.com/hackers365/gtcrn-denoise-go
```

The package vendors the modified sherpa-onnx C API header and sherpa native
libraries under `denoise/native`. `onnxruntime` is not vendored and must be
provided by the build/runtime environment.

If `onnxruntime` is installed in a standard linker path, no extra flags are
needed. Otherwise pass its library directory when building:

```bash
CGO_ENABLED=1 CGO_LDFLAGS="-L/path/to/onnxruntime/lib" go build ./...
```

At runtime, make sure the ONNX Runtime library can be found. On Linux:

```bash
export LD_LIBRARY_PATH=/path/to/onnxruntime/lib:$LD_LIBRARY_PATH
```

On macOS:

```bash
export DYLD_LIBRARY_PATH=/path/to/onnxruntime/lib:$DYLD_LIBRARY_PATH
```

On Windows, prefer putting the matching `onnxruntime.dll` beside the final
executable. A stale `C:\Windows\System32\onnxruntime.dll` can be loaded before
entries from `PATH`, causing model load failures even when a newer runtime is in
`PATH`.

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

## Native libraries

Current vendored native assets:

- `denoise/native/include/c-api.h`
- `denoise/native/lib/linux_amd64/libsherpa-onnx-c-api.so`
- `denoise/native/lib/linux_arm64/libsherpa-onnx-c-api.so`
- `denoise/native/lib/darwin_amd64/libsherpa-onnx-c-api.dylib`
- `denoise/native/lib/darwin_arm64/libsherpa-onnx-c-api.dylib`
- `denoise/native/lib/windows_amd64/sherpa-onnx-c-api.dll`
- `denoise/native/lib/windows_amd64/sherpa-onnx-c-api.lib`

Not vendored:

- `onnxruntime`
- GTCRN model files such as `gtcrn_simple.onnx`

Keep `onnxruntime` outside the repo and provide it through runtime search paths
and linker flags when it is not installed in a standard location.
