# Task 2: 表达式求值器

## 题目描述

实现一个数学表达式求值器，支持基本运算、括号、变量和自定义函数调用。

## 功能要求

### 基础功能
1. **基本运算**: 支持+、-、*、/四则运算
2. **括号支持**: 支持括号改变运算优先级
3. **数值类型**: 支持整数和浮点数
4. **空格处理**: 自动忽略表达式中的空格

### 高级功能
1. **变量支持**: 支持变量定义和使用
2. **函数调用**: 支持内置函数和自定义函数
3. **错误处理**: 提供详细的错误信息和位置提示
4. **表达式验证**: 验证表达式语法正确性

### 内置函数
- `sin(x)`, `cos(x)`, `tan(x)` - 三角函数
- `sqrt(x)` - 平方根
- `abs(x)` - 绝对值
- `pow(x, y)` - 幂运算
- `max(a, b)`, `min(a, b)` - 最大最小值

## 技术要求

### 解析方法
- 使用递归下降解析器（Recursive Descent Parser）
- 或使用Shunting-yard算法
- 支持运算符优先级和结合性

### 错误处理
- 语法错误：括号不匹配、运算符使用错误等
- 语义错误：变量未定义、函数参数错误等
- 运行时错误：除零、函数计算错误等

### 性能要求
- 单次表达式求值时间<10ms
- 支持表达式预编译和重复求值

## 接口定义

```kotlin
interface ExpressionEvaluator {
    // 设置变量
    fun setVariable(name: String, value: Double)
    
    // 获取变量
    fun getVariable(name: String): Double?
    
    // 注册自定义函数
    fun registerFunction(name: String, function: (List<Double>) -> Double)
    
    // 求值表达式
    fun evaluate(expression: String): Double
    
    // 验证表达式
    fun validate(expression: String): ValidationResult
    
    // 预编译表达式
    fun compile(expression: String): CompiledExpression
}

sealed class ValidationResult {
    object Valid : ValidationResult()
    data class Invalid(val error: String, val position: Int) : ValidationResult()
}

interface CompiledExpression {
    fun evaluate(variables: Map<String, Double> = emptyMap()): Double
}

class ExpressionEvaluatorImpl : ExpressionEvaluator {
    // TODO: 实现细节
}
```

## 测试用例

### 基础表达式测试

```kotlin
@Test
fun testBasicArithmetic() {
    val evaluator = ExpressionEvaluatorImpl()
    
    assertEquals(5.0, evaluator.evaluate("2 + 3"), 0.0001)
    assertEquals(6.0, evaluator.evaluate("2 * 3"), 0.0001)
    assertEquals(-1.0, evaluator.evaluate("2 - 3"), 0.0001)
    assertEquals(2.0, evaluator.evaluate("6 / 3"), 0.0001)
}

@Test
fun testOperatorPrecedence() {
    val evaluator = ExpressionEvaluatorImpl()
    
    // 乘除优先于加减
    assertEquals(14.0, evaluator.evaluate("2 + 3 * 4"), 0.0001)
    assertEquals(10.0, evaluator.evaluate("2 * 3 + 4"), 0.0001)
    
    // 括号改变优先级
    assertEquals(20.0, evaluator.evaluate("(2 + 3) * 4"), 0.0001)
    assertEquals(14.0, evaluator.evaluate("2 * (3 + 4)"), 0.0001)
}

@Test
fun testComplexExpressions() {
    val evaluator = ExpressionEvaluatorImpl()
    
    assertEquals(7.0, evaluator.evaluate("2 + 3 * 4 - 9 / 3"), 0.0001)
    assertEquals(10.0, evaluator.evaluate("((2 + 3) * 2)"), 0.0001)
    assertEquals(2.5, evaluator.evaluate("10 / (2 + 2)"), 0.0001)
}
```

### 变量测试

```kotlin
@Test
fun testVariables() {
    val evaluator = ExpressionEvaluatorImpl()
    
    evaluator.setVariable("x", 10.0)
    evaluator.setVariable("y", 20.0)
    
    assertEquals(30.0, evaluator.evaluate("x + y"), 0.0001)
    assertEquals(200.0, evaluator.evaluate("x * y"), 0.0001)
    assertEquals(15.0, evaluator.evaluate("(x + y) / 2"), 0.0001)
}

@Test
fun testUndefinedVariable() {
    val evaluator = ExpressionEvaluatorImpl()
    
    assertThrows<UndefinedVariableException> {
        evaluator.evaluate("z + 1")
    }
}
```

### 函数测试

```kotlin
@Test
fun testBuiltinFunctions() {
    val evaluator = ExpressionEvaluatorImpl()
    
    assertEquals(0.0, evaluator.evaluate("sin(0)"), 0.0001)
    assertEquals(1.0, evaluator.evaluate("cos(0)"), 0.0001)
    assertEquals(2.0, evaluator.evaluate("sqrt(4)"), 0.0001)
    assertEquals(5.0, evaluator.evaluate("abs(-5)"), 0.0001)
    assertEquals(8.0, evaluator.evaluate("pow(2, 3)"), 0.0001)
    assertEquals(10.0, evaluator.evaluate("max(5, 10)"), 0.0001)
}

@Test
fun testCustomFunctions() {
    val evaluator = ExpressionEvaluatorImpl()
    
    // 注册自定义函数：计算圆面积
    evaluator.registerFunction("circleArea") { args ->
        val radius = args[0]
        Math.PI * radius * radius
    }
    
    assertEquals(Math.PI * 4, evaluator.evaluate("circleArea(2)"), 0.0001)
    
    // 注册多参数函数
    evaluator.registerFunction("add") { args ->
        args.sum()
    }
    
    assertEquals(6.0, evaluator.evaluate("add(1, 2, 3)"), 0.0001)
}

@Test
fun testNestedFunctions() {
    val evaluator = ExpressionEvaluatorImpl()
    
    assertEquals(2.0, evaluator.evaluate("sqrt(abs(-4))"), 0.0001)
    assertEquals(0.0, evaluator.evaluate("sin(max(0, 0))"), 0.0001)
}
```

### 错误处理测试

```kotlin
@Test
fun testSyntaxErrors() {
    val evaluator = ExpressionEvaluatorImpl()
    
    // 括号不匹配
    assertThrows<SyntaxException> {
        evaluator.evaluate("(2 + 3")
    }
    
    // 运算符错误
    assertThrows<SyntaxException> {
        evaluator.evaluate("2 + + 3")
    }
    
    // 表达式不完整
    assertThrows<SyntaxException> {
        evaluator.evaluate("2 +")
    }
}

@Test
fun testRuntimeErrors() {
    val evaluator = ExpressionEvaluatorImpl()
    
    // 除零错误
    assertThrows<DivisionByZeroException> {
        evaluator.evaluate("1 / 0")
    }
    
    // 函数参数错误
    assertThrows<InvalidArgumentException> {
        evaluator.evaluate("sqrt(-1)")
    }
}

@Test
fun testValidation() {
    val evaluator = ExpressionEvaluatorImpl()
    
    // 有效表达式
    assertTrue(evaluator.validate("2 + 3") is ValidationResult.Valid)
    
    // 无效表达式
    val result = evaluator.validate("2 + + 3")
    assertTrue(result is ValidationResult.Invalid)
    assertNotNull((result as ValidationResult.Invalid).error)
    assertTrue(result.position >= 0)
}
```

### 性能测试

```kotlin
@Test
fun testPerformance() {
    val evaluator = ExpressionEvaluatorImpl()
    val expression = "2 + 3 * 4 - 5 / 2 + sin(0.5) * cos(0.3)"
    
    val startTime = System.currentTimeMillis()
    
    repeat(1000) {
        evaluator.evaluate(expression)
    }
    
    val endTime = System.currentTimeMillis()
    val avgTime = (endTime - startTime) / 1000.0
    
    // 单次求值应<10ms
    assertTrue(avgTime < 10.0, "Average evaluation time: ${avgTime}ms")
}

@Test
fun testCompiledExpression() {
    val evaluator = ExpressionEvaluatorImpl()
    val compiled = evaluator.compile("x * 2 + y")
    
    // 多次求值
    val result1 = compiled.evaluate(mapOf("x" to 5.0, "y" to 3.0))
    assertEquals(13.0, result1, 0.0001)
    
    val result2 = compiled.evaluate(mapOf("x" to 10.0, "y" to 7.0))
    assertEquals(27.0, result2, 0.0001)
}
```

## 验收标准

### 功能验收
- [ ] 基础四则运算正确
- [ ] 运算符优先级正确
- [ ] 括号处理正确
- [ ] 变量功能正常
- [ ] 内置函数正常
- [ ] 自定义函数正常

### 错误处理验收
- [ ] 语法错误检测准确
- [ ] 错误信息清晰明确
- [ ] 错误位置定位准确
- [ ] 异常类型分类合理

### 性能验收
- [ ] 单次求值<10ms
- [ ] 预编译功能正常
- [ ] 内存使用合理

### 代码质量
- [ ] 解析器结构清晰
- [ ] 代码可读性强
- [ ] 测试覆盖率≥80%
- [ ] 包含详细注释

## 提示

1. 建议使用递归下降解析器，结构清晰易扩展
2. 运算符优先级（从低到高）：+ - < * / < 一元运算符 < 函数调用
3. 使用AST（抽象语法树）表示表达式结构
4. 错误处理要包含位置信息，便于定位问题
5. 考虑使用访问者模式遍历AST

## 参考资料

- 递归下降解析器：https://en.wikipedia.org/wiki/Recursive_descent_parser
- Shunting-yard算法：https://en.wikipedia.org/wiki/Shunting-yard_algorithm
- 表达式求值：https://kotlinlang.org/docs/grammar.html#expressions
