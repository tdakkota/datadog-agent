package pb

var metahook = func(_, v string) string { return v }

// RegisterMetaHook ...
func RegisterMetaHook(hook func(k, v string) string) {
	metahook = hook
}
