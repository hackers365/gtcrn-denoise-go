# GTCRN denoise Go

This project contains a small integration package in `./denoise`.
It calls the modified sherpa-onnx C API directly through cgo, so it does not
depend on `github.com/k2-fsa/sherpa-onnx-go` or the local `scripts/go/_internal`
wrapper.

## Install

```bash
go get github.com/hackers365/gtcrn-denoise-go
```

The package vendors the modified sherpa-onnx C API header and sherpa-related
Linux amd64 static libraries under `denoise/native`. `onnxruntime` is not
vendored and must be provided by the build/runtime environment.

If `onnxruntime` is installed in a standard linker path, no extra flags are
needed. Otherwise pass its library directory when building:

```bash
CGO_ENABLED=1 CGO_LDFLAGS="-L/path/to/onnxruntime/lib" go build ./...
```

At runtime, make sure `libonnxruntime.so` can be found, for example:

```bash
export LD_LIBRARY_PATH=/path/to/onnxruntime/lib:$LD_LIBRARY_PATH
```

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

## Native libraries

Current vendored native assets:

- `denoise/native/include/c-api.h`
- `denoise/native/lib/linux_amd64/*.a`

Not vendored:

- `onnxruntime`
- GTCRN model files such as `gtcrn_simple.onnx`

Windows support uses `denoise/native/lib/windows_amd64`. Put the sherpa-onnx
Windows import library/DLL there when available. Keep `onnxruntime.dll` and
`onnxruntime.lib` outside the repo and provide them through `PATH`/linker flags.
