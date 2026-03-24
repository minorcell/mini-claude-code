package tools

// DefaultRegistry 返回带默认拦截器和内置工具的注册中心。
func DefaultRegistry(workspace string) *Registry {
	registry := NewRegistry(
		WorkspaceInterceptor{Root: workspace},
		ShellSafetyInterceptor{},
	)
	registry.MustRegister(NewBashTool(workspace))
	registry.MustRegister(NewReadTool(workspace))
	registry.MustRegister(NewWriteTool(workspace))
	registry.MustRegister(NewEditTool(workspace))
	registry.MustRegister(NewListTool(workspace))
	registry.MustRegister(NewGlobTool(workspace))
	registry.MustRegister(NewGrepTool(workspace))
	registry.MustRegister(NewTodoTool())
	registry.MustRegister(NewWebFetchTool())
	return registry
}
