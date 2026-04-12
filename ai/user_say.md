# 用户需求：文件写入/编辑工具增加语法检查功能

## 背景

AgentPlus在创建文件、编辑文件后，由于模型能力的原因，可能会出现基础语法错误。需要为文件写入、编辑工具增加操作完毕后的文件语法检查功能，并要求模型文件的写入和编辑尽可能使用工具而不是命令行防止绕过。

## 核心需求

### 1. 语法检查功能

要求程序在文件写入或编辑操作完成后，立即根据文件扩展名进行语法检查。

需要支持的文件类型及对应扩展名：
- **Java** (.java) - 支持至Java 26语法
- **Kotlin** (.kt, .kts) - 支持至Kotlin 2.x语法
- **JavaScript** (.js, .mjs) - 支持现代ES语法
- **TypeScript** (.ts, .tsx) - 支持最新TS语法
- **Python** (.py, .pyw) - 支持Python 3语法
- **Shell** (.sh, .bash) - Shell脚本语法
- **PowerShell** (.ps1, .psm1) - PowerShell脚本语法
- **Go** (.go) - Go语言语法
- **Rust** (.rs) - Rust语言语法
- **HTML** (.html, .htm) - HTML结构检查
- **HTML+JS** - HTML中嵌入的JavaScript语法检查
- **BAT** (.bat, .cmd) - Windows批处理语法
- **Gradle** (.gradle, .gradle.kts) - Gradle构建脚本语法
- **XML** (.xml, .xsd, .xsl, .xslt, .svg, .pom) - XML结构检查
- **YAML** (.yaml, .yml) - YAML语法检查
- **JSON** (.json) - JSON语法检查
- **CSS** (.css) - CSS语法检查
- **SQL** (.sql) - SQL基础语法检查
- **Properties** (.properties) - Java Properties格式检查
- ** TOML** (.toml) - TOML格式检查

### 2. 语法检查规范

- **必须是真正的语法检查（AST解析），而不是正则匹配**
- 支持最新版本的语言语法
- 语法检查应该能发现：语法错误、未闭合的括号/引号/标签、无效的标识符、缺少分号等基础语法问题
- 检查结果应包含：错误类型、行号/位置、错误描述、修正建议

### 3. 工具行为变更

- 模型调用write_file或edit_file工具结束后，工具立即根据文件扩展名进行语法检查
- 语法检查结果附加在工具返回结果中
- 工具的Description说明中需要增加相关机制描述，协助模型理解返回结果
- 当发现语法错误时，应提醒模型接下来应该继续完善文件避免存在语法错误

### 4. edit_file工具

当前项目只有write_file（整体覆盖写入），需要新增edit_file工具：
- 支持基于行号的编辑（指定起始行和结束行进行替换）
- 支持字符串替换编辑（指定old_string和new_string进行替换）
- 编辑完成后同样进行语法检查

### 5. 防绕过机制

工具Description中应强调模型必须使用工具进行文件操作，不建议通过命令行方式绕过。
