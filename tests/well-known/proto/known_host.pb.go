//go:build !tinygo.wasm

// Code generated by protoc-gen-go-plugin. DO NOT EDIT.
// versions:
// 	protoc-gen-go-plugin v0.1.0
// 	protoc               v3.21.12
// source: tests/well-known/proto/known.proto

package proto

import (
	context "context"
	errors "errors"
	fmt "fmt"
	emptypb "github.com/knqyf263/go-plugin/types/known/emptypb"
	wazero "github.com/tetratelabs/wazero"
	api "github.com/tetratelabs/wazero/api"
	sys "github.com/tetratelabs/wazero/sys"
	os "os"
)

const KnownTypesTestPluginAPIVersion = 1

type KnownTypesTestPlugin struct {
	newRuntime   func(context.Context) (wazero.Runtime, error)
	moduleConfig wazero.ModuleConfig
}

func NewKnownTypesTestPlugin(ctx context.Context, opts ...wazeroConfigOption) (*KnownTypesTestPlugin, error) {
	o := &WazeroConfig{
		newRuntime:   DefaultWazeroRuntime(),
		moduleConfig: wazero.NewModuleConfig(),
	}

	for _, opt := range opts {
		opt(o)
	}

	return &KnownTypesTestPlugin{
		newRuntime:   o.newRuntime,
		moduleConfig: o.moduleConfig,
	}, nil
}

type knownTypesTest interface {
	Close(ctx context.Context) error
	KnownTypesTest
}

func (p *KnownTypesTestPlugin) Load(ctx context.Context, pluginPath string) (knownTypesTest, error) {
	b, err := os.ReadFile(pluginPath)
	if err != nil {
		return nil, err
	}

	// Create a new runtime so that multiple modules will not conflict
	r, err := p.newRuntime(ctx)
	if err != nil {
		return nil, err
	}

	// Compile the WebAssembly module using the default configuration.
	code, err := r.CompileModule(ctx, b)
	if err != nil {
		return nil, err
	}

	// InstantiateModule runs the "_start" function, WASI's "main".
	module, err := r.InstantiateModule(ctx, code, p.moduleConfig)
	if err != nil {
		// Note: Most compilers do not exit the module after running "_start",
		// unless there was an Error. This allows you to call exported functions.
		if exitErr, ok := err.(*sys.ExitError); ok && exitErr.ExitCode() != 0 {
			return nil, fmt.Errorf("unexpected exit_code: %d", exitErr.ExitCode())
		} else if !ok {
			return nil, err
		}
	}

	// Compare API versions with the loading plugin
	apiVersion := module.ExportedFunction("known_types_test_api_version")
	if apiVersion == nil {
		return nil, errors.New("known_types_test_api_version is not exported")
	}
	results, err := apiVersion.Call(ctx)
	if err != nil {
		return nil, err
	} else if len(results) != 1 {
		return nil, errors.New("invalid known_types_test_api_version signature")
	}
	if results[0] != KnownTypesTestPluginAPIVersion {
		return nil, fmt.Errorf("API version mismatch, host: %d, plugin: %d", KnownTypesTestPluginAPIVersion, results[0])
	}

	test := module.ExportedFunction("known_types_test_test")
	if test == nil {
		return nil, errors.New("known_types_test_test is not exported")
	}

	malloc := module.ExportedFunction("malloc")
	if malloc == nil {
		return nil, errors.New("malloc is not exported")
	}

	free := module.ExportedFunction("free")
	if free == nil {
		return nil, errors.New("free is not exported")
	}
	return &knownTypesTestPlugin{
		runtime: r,
		module:  module,
		malloc:  malloc,
		free:    free,
		test:    test,
	}, nil
}

func (p *knownTypesTestPlugin) Close(ctx context.Context) (err error) {
	if r := p.runtime; r != nil {
		r.Close(ctx)
	}
	return
}

type knownTypesTestPlugin struct {
	runtime wazero.Runtime
	module  api.Module
	malloc  api.Function
	free    api.Function
	test    api.Function
}

func (p *knownTypesTestPlugin) Test(ctx context.Context, request *Request) (*Response, error) {
	data, err := request.MarshalVT()
	if err != nil {
		return nil, err
	}
	dataSize := uint64(len(data))

	var dataPtr uint64
	// If the input data is not empty, we must allocate the in-Wasm memory to store it, and pass to the plugin.
	if dataSize != 0 {
		results, err := p.malloc.Call(ctx, dataSize)
		if err != nil {
			return nil, err
		}
		dataPtr = results[0]
		// This pointer is managed by TinyGo, but TinyGo is unaware of external usage.
		// So, we have to free it when finished
		defer p.free.Call(ctx, dataPtr)

		// The pointer is a linear memory offset, which is where we write the name.
		if !p.module.Memory().Write(uint32(dataPtr), data) {
			return nil, fmt.Errorf("Memory.Write(%d, %d) out of range of memory size %d", dataPtr, dataSize, p.module.Memory().Size())
		}
	}

	ptrSize, err := p.test.Call(ctx, dataPtr, dataSize)
	if err != nil {
		return nil, err
	}

	resPtr := uint32(ptrSize[0] >> 32)
	resSize := uint32(ptrSize[0])
	var isErrResponse bool
	if (resSize & (1 << 31)) > 0 {
		isErrResponse = true
		resSize &^= (1 << 31)
	}

	// We don't need the memory after deserialization: make sure it is freed.
	if resPtr != 0 {
		defer p.free.Call(ctx, uint64(resPtr))
	}

	// The pointer is a linear memory offset, which is where we write the name.
	bytes, ok := p.module.Memory().Read(resPtr, resSize)
	if !ok {
		return nil, fmt.Errorf("Memory.Read(%d, %d) out of range of memory size %d",
			resPtr, resSize, p.module.Memory().Size())
	}

	if isErrResponse {
		return nil, errors.New(string(bytes))
	}

	response := new(Response)
	if err = response.UnmarshalVT(bytes); err != nil {
		return nil, err
	}

	return response, nil
}

const EmptyTestPluginAPIVersion = 1

type EmptyTestPlugin struct {
	newRuntime   func(context.Context) (wazero.Runtime, error)
	moduleConfig wazero.ModuleConfig
}

func NewEmptyTestPlugin(ctx context.Context, opts ...wazeroConfigOption) (*EmptyTestPlugin, error) {
	o := &WazeroConfig{
		newRuntime:   DefaultWazeroRuntime(),
		moduleConfig: wazero.NewModuleConfig(),
	}

	for _, opt := range opts {
		opt(o)
	}

	return &EmptyTestPlugin{
		newRuntime:   o.newRuntime,
		moduleConfig: o.moduleConfig,
	}, nil
}

type emptyTest interface {
	Close(ctx context.Context) error
	EmptyTest
}

func (p *EmptyTestPlugin) Load(ctx context.Context, pluginPath string) (emptyTest, error) {
	b, err := os.ReadFile(pluginPath)
	if err != nil {
		return nil, err
	}

	// Create a new runtime so that multiple modules will not conflict
	r, err := p.newRuntime(ctx)
	if err != nil {
		return nil, err
	}

	// Compile the WebAssembly module using the default configuration.
	code, err := r.CompileModule(ctx, b)
	if err != nil {
		return nil, err
	}

	// InstantiateModule runs the "_start" function, WASI's "main".
	module, err := r.InstantiateModule(ctx, code, p.moduleConfig)
	if err != nil {
		// Note: Most compilers do not exit the module after running "_start",
		// unless there was an Error. This allows you to call exported functions.
		if exitErr, ok := err.(*sys.ExitError); ok && exitErr.ExitCode() != 0 {
			return nil, fmt.Errorf("unexpected exit_code: %d", exitErr.ExitCode())
		} else if !ok {
			return nil, err
		}
	}

	// Compare API versions with the loading plugin
	apiVersion := module.ExportedFunction("empty_test_api_version")
	if apiVersion == nil {
		return nil, errors.New("empty_test_api_version is not exported")
	}
	results, err := apiVersion.Call(ctx)
	if err != nil {
		return nil, err
	} else if len(results) != 1 {
		return nil, errors.New("invalid empty_test_api_version signature")
	}
	if results[0] != EmptyTestPluginAPIVersion {
		return nil, fmt.Errorf("API version mismatch, host: %d, plugin: %d", EmptyTestPluginAPIVersion, results[0])
	}

	donothing := module.ExportedFunction("empty_test_do_nothing")
	if donothing == nil {
		return nil, errors.New("empty_test_do_nothing is not exported")
	}

	malloc := module.ExportedFunction("malloc")
	if malloc == nil {
		return nil, errors.New("malloc is not exported")
	}

	free := module.ExportedFunction("free")
	if free == nil {
		return nil, errors.New("free is not exported")
	}
	return &emptyTestPlugin{
		runtime:   r,
		module:    module,
		malloc:    malloc,
		free:      free,
		donothing: donothing,
	}, nil
}

func (p *emptyTestPlugin) Close(ctx context.Context) (err error) {
	if r := p.runtime; r != nil {
		r.Close(ctx)
	}
	return
}

type emptyTestPlugin struct {
	runtime   wazero.Runtime
	module    api.Module
	malloc    api.Function
	free      api.Function
	donothing api.Function
}

func (p *emptyTestPlugin) DoNothing(ctx context.Context, request *emptypb.Empty) (*emptypb.Empty, error) {
	data, err := request.MarshalVT()
	if err != nil {
		return nil, err
	}
	dataSize := uint64(len(data))

	var dataPtr uint64
	// If the input data is not empty, we must allocate the in-Wasm memory to store it, and pass to the plugin.
	if dataSize != 0 {
		results, err := p.malloc.Call(ctx, dataSize)
		if err != nil {
			return nil, err
		}
		dataPtr = results[0]
		// This pointer is managed by TinyGo, but TinyGo is unaware of external usage.
		// So, we have to free it when finished
		defer p.free.Call(ctx, dataPtr)

		// The pointer is a linear memory offset, which is where we write the name.
		if !p.module.Memory().Write(uint32(dataPtr), data) {
			return nil, fmt.Errorf("Memory.Write(%d, %d) out of range of memory size %d", dataPtr, dataSize, p.module.Memory().Size())
		}
	}

	ptrSize, err := p.donothing.Call(ctx, dataPtr, dataSize)
	if err != nil {
		return nil, err
	}

	resPtr := uint32(ptrSize[0] >> 32)
	resSize := uint32(ptrSize[0])
	var isErrResponse bool
	if (resSize & (1 << 31)) > 0 {
		isErrResponse = true
		resSize &^= (1 << 31)
	}

	// We don't need the memory after deserialization: make sure it is freed.
	if resPtr != 0 {
		defer p.free.Call(ctx, uint64(resPtr))
	}

	// The pointer is a linear memory offset, which is where we write the name.
	bytes, ok := p.module.Memory().Read(resPtr, resSize)
	if !ok {
		return nil, fmt.Errorf("Memory.Read(%d, %d) out of range of memory size %d",
			resPtr, resSize, p.module.Memory().Size())
	}

	if isErrResponse {
		return nil, errors.New(string(bytes))
	}

	response := new(emptypb.Empty)
	if err = response.UnmarshalVT(bytes); err != nil {
		return nil, err
	}

	return response, nil
}
