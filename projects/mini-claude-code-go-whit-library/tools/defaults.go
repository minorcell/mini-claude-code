package tools

func DefaultRegistry(workspace string) *Registry {
	registry := NewRegistry(
		WorkspaceInterceptor{Root: workspace},
		ShellSafetyInterceptor{},
	)
	registry.MustRegister(NewBashTool(workspace))
	registry.MustRegister(NewFileSystemTool(workspace))
	registry.MustRegister(NewWebFetchTool())
	return registry
}
