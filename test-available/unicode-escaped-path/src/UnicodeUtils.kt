package test

/**
 * Unicode-Escaped 编码/解码工具函数
 */

/**
 * 将字符串转换为 Unicode-escaped 格式
 * 例如："中文" -> "\u4e2d\u6587"
 */
fun String.toUnicodeEscaped(): String {
    return this.map { char ->
        if (char.code > 127) {
            // 非 ASCII 字符转换为 \uXXXX 格式
            "\\u${char.code.toString(16).padStart(4, '0')}"
        } else {
            // ASCII 字符保持不变
            char.toString()
        }
    }.joinToString("")
}

/**
 * 将 Unicode-escaped 格式解码为原始字符串
 * 例如："\u4e2d\u6587" -> "中文"
 */
fun String.fromUnicodeEscaped(): String {
    val regex = Regex("\\\\u([0-9a-fA-F]{4})")
    return regex.replace(this) { matchResult ->
        val unicode = matchResult.groupValues[1].toIntOrNull(16)
        if (unicode != null) {
            unicode.toChar().toString()
        } else {
            matchResult.value // 如果转换失败，保持原样
        }
    }
}
