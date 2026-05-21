# GTCRN denoise Go

This project contains a small integration package in `./denoise`.
It calls the modified sherpa-onnx C API directly through cgo, so it does not
depend on `github.com/k2-fsa/sherpa-onnx-go` or the local `scripts/go/_internal`
wrapper.

```go
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

Linux builds expect the sibling `../sherpa-onnx` tree to contain the modified
C API header and matching native libraries under `../sherpa-onnx/build/lib`.
Both static and shared Linux library layouts are supported when the required
sherpa-onnx build outputs are present in that directory.
Windows builds expect `sherpa-onnx-c-api.dll`, `onnxruntime.dll`, and their
import libraries under `./denoise/native/lib/windows_amd64`.
