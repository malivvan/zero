package wasmtest

import (
	"context"

	"github.com/malivvan/zero/zerg"
	"github.com/malivvan/zero/zerg/types"
	"github.com/tetratelabs/wazero/api"
)

func Call[R types.Param[R], T any](fn wazergo.Function[T], ctx context.Context, module api.Module, this T, args ...types.Result) (ret R) {
	malloc = 0

	stack := make([]uint64, max(fn.NumParams(), fn.NumResults()))
	memory := module.Memory()
	offset := 0

	for _, arg := range args {
		arg.StoreValue(memory, stack[offset:])
		offset += len(arg.ValueTypes())
	}

	fn.Func(this, ctx, module, stack)
	return ret.LoadValue(memory, stack)
}

func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}
