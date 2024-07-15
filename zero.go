package zero

import (
	"context"
	"fmt"
	"github.com/malivvan/zero/wasi"
	"github.com/malivvan/zero/wasi/imports"
	"github.com/malivvan/zero/wasi/imports/wasi_http"
	"github.com/tetratelabs/wazero"
	"net/http"
	"os"
	"path/filepath"
)

type Config struct {
	envInherit       bool
	Envs             stringList
	Dirs             stringList
	Listens          stringList
	Dials            stringList
	DNSServer        string
	SocketExt        string
	ProfAddr         string
	WasiHttp         string
	WasiHttpAddr     string
	WasiHttpPath     string
	Trace            bool
	TracerStringSize int
	NonBlockingStdio bool
	Version          bool
	MaxOpenFiles     int
	MaxOpenDirs      int
}

func Run(cfg Config, wasmFile string, args []string) error {
	if cfg.WasiHttp == "" {
		cfg.WasiHttp = "auto"
	}

	wasmName := filepath.Base(wasmFile)
	wasmCode, err := os.ReadFile(wasmFile)
	if err != nil {
		return fmt.Errorf("could not read WASM file '%s': %w", wasmFile, err)
	}

	if len(args) > 0 && args[0] == "--" {
		args = args[1:]
	}
	if cfg.ProfAddr != "" {
		go http.ListenAndServe(cfg.ProfAddr, nil)
	}

	ctx := context.Background()
	runtime := wazero.NewRuntime(ctx)
	defer runtime.Close(ctx)

	wasmModule, err := runtime.CompileModule(ctx, wasmCode)
	if err != nil {
		return err
	}
	defer wasmModule.Close(ctx)

	builder := imports.NewBuilder().
		WithName(wasmName).
		WithArgs(args...).
		WithEnv(cfg.Envs...).
		WithDirs(cfg.Dirs...).
		WithListens(cfg.Listens...).
		WithDials(cfg.Dials...).
		WithNonBlockingStdio(cfg.NonBlockingStdio).
		WithSocketsExtension(cfg.SocketExt, wasmModule).
		WithTracer(cfg.Trace, os.Stderr, wasi.WithTracerStringSize(cfg.TracerStringSize)).
		WithMaxOpenFiles(cfg.MaxOpenFiles).
		WithMaxOpenDirs(cfg.MaxOpenDirs)

	var system wasi.System
	ctx, system, err = builder.Instantiate(ctx, runtime)
	if err != nil {
		return err
	}
	defer system.Close(ctx)

	importWasi := false
	var wasiHTTP *wasi_http.WasiHTTP = nil
	switch cfg.WasiHttp {
	case "auto":
		importWasi = wasi_http.DetectWasiHttp(wasmModule)
	case "v1":
		importWasi = true
	case "none":
		importWasi = false
	default:
		return fmt.Errorf("invalid value for -http '%v', expected 'auto', 'v1' or 'none'", cfg.WasiHttp)
	}
	if importWasi {
		wasiHTTP = wasi_http.MakeWasiHTTP()
		if err := wasiHTTP.Instantiate(ctx, runtime); err != nil {
			return err
		}
	}

	instance, err := runtime.InstantiateModule(ctx, wasmModule, wazero.NewModuleConfig())
	if err != nil {
		return err
	}
	if len(cfg.WasiHttpAddr) > 0 {
		handler := wasiHTTP.MakeHandler(ctx, instance)
		http.Handle(cfg.WasiHttpPath, handler)
		return http.ListenAndServe(cfg.WasiHttpAddr, nil)
	}
	return instance.Close(ctx)
}

type stringList []string

func (s stringList) String() string {
	return fmt.Sprintf("%v", []string(s))
}

func (s *stringList) Set(value string) error {
	*s = append(*s, value)
	return nil
}
