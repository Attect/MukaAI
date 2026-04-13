package syntax

// RegisterAllCheckers 注册所有可用的语法检查器到调度器
func RegisterAllCheckers(d *Dispatcher) {
	// Go原生解析器（无外部依赖）
	d.RegisterChecker(NewJSONChecker())
	d.RegisterChecker(NewYAMLChecker())
	d.RegisterChecker(NewXMLChecker())
	d.RegisterChecker(NewHTMLChecker())
	d.RegisterChecker(NewGoChecker())

	// 原生解析器（第三方库依赖，编译到二进制中）
	d.RegisterChecker(NewTOMLChecker())
	d.RegisterChecker(NewCSSChecker())

	// 原生解析器（Go标准库实现）
	d.RegisterChecker(NewSQLChecker())
	d.RegisterChecker(NewPropertiesChecker())

	// 外部工具检查器（可选依赖，不可用时自动降级）
	d.RegisterChecker(NewExternalChecker())
}
