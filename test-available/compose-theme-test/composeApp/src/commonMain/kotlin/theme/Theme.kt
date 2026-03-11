package theme

import androidx.compose.foundation.isSystemInDarkTheme
import androidx.compose.material3.MaterialTheme
import androidx.compose.material3.darkColorScheme
import androidx.compose.material3.lightColorScheme
import androidx.compose.runtime.Composable
import androidx.compose.runtime.CompositionLocalProvider
import androidx.compose.runtime.collectAsState
import androidx.compose.runtime.compositionLocalOf
import androidx.compose.runtime.remember
import androidx.compose.ui.graphics.Color
import com.russhwolf.settings.ObservableSettings
import com.russhwolf.settings.Settings
import com.russhwolf.settings.coroutines.FlowSettings
import com.russhwolf.settings.coroutines.toFlowSettings
import kotlinx.coroutines.flow.Flow
import kotlinx.coroutines.flow.map
import kotlinx.serialization.Serializable
import kotlinx.serialization.json.Json

/**
 * 主题模式
 */
@Serializable
enum class ThemeMode {
    LIGHT,
    DARK,
    AUTO
}

/**
 * 主题配置
 */
@Serializable
data class ThemeConfig(
    val mode: ThemeMode = ThemeMode.AUTO
)

/**
 * Apple风格颜色方案
 */
private val LightColors = lightColorScheme(
    primary = Color(0xFF007AFF),
    onPrimary = Color(0xFFFFFFFF),
    secondary = Color(0xFF5856D6),
    onSecondary = Color(0xFFFFFFFF),
    background = Color(0xFFF5F5F7),
    onBackground = Color(0xFF1D1D1F),
    surface = Color(0xFFFFFFFF),
    onSurface = Color(0xFF1D1D1F),
    error = Color(0xFFFF3B30),
    onError = Color(0xFFFFFFFF),
    outline = Color(0xFFE5E5EA),
    surfaceVariant = Color(0xFFF2F2F7),
    onSurfaceVariant = Color(0xFF8E8E93)
)

private val DarkColors = darkColorScheme(
    primary = Color(0xFF0A84FF),
    onPrimary = Color(0xFFFFFFFF),
    secondary = Color(0xFF5E5CE6),
    onSecondary = Color(0xFFFFFFFF),
    background = Color(0xFF000000),
    onBackground = Color(0xFFFFFFFF),
    surface = Color(0xFF1C1C1E),
    onSurface = Color(0xFFFFFFFF),
    error = Color(0xFFFF453A),
    onError = Color(0xFFFFFFFF),
    outline = Color(0xFF38383A),
    surfaceVariant = Color(0xFF2C2C2E),
    onSurfaceVariant = Color(0xFF8E8E93)
)

/**
 * 主题管理器
 */
class ThemeManager(private val settings: Settings) {
    private val flowSettings: FlowSettings = (settings as ObservableSettings).toFlowSettings()
    private val json = Json { ignoreUnknownKeys = true }
    
    /**
     * 当前主题配置流
     */
    val themeConfig: Flow<ThemeConfig> = flowSettings
        .getStringOrNullFlow(KEY_THEME_CONFIG)
        .map { jsonString ->
            jsonString?.let {
                try {
                    json.decodeFromString<ThemeConfig>(it)
                } catch (e: Exception) {
                    ThemeConfig()
                }
            } ?: ThemeConfig()
        }
    
    /**
     * 获取当前有效的颜色方案
     */
    @Composable
    fun getColorScheme(themeConfig: ThemeConfig): androidx.compose.material3.ColorScheme {
        val isSystemDark = isSystemInDarkTheme()
        return when (themeConfig.mode) {
            ThemeMode.LIGHT -> LightColors
            ThemeMode.DARK -> DarkColors
            ThemeMode.AUTO -> if (isSystemDark) DarkColors else LightColors
        }
    }
    
    /**
     * 设置主题模式
     */
    suspend fun setThemeMode(mode: ThemeMode) {
        val currentJson = settings.getStringOrNull(KEY_THEME_CONFIG)
        val current = currentJson?.let {
            try {
                json.decodeFromString<ThemeConfig>(it)
            } catch (e: Exception) {
                ThemeConfig()
            }
        } ?: ThemeConfig()
        
        val newConfig = current.copy(mode = mode)
        settings.putString(KEY_THEME_CONFIG, json.encodeToString(ThemeConfig.serializer(), newConfig))
    }
    
    companion object {
        private const val KEY_THEME_CONFIG = "theme_config"
    }
}

/**
 * Local主题管理器
 */
val LocalThemeManager = compositionLocalOf<ThemeManager> {
    error("ThemeManager not provided")
}

/**
 * 应用主题
 */
@Composable
fun AppTheme(
    themeManager: ThemeManager,
    content: @Composable () -> Unit
) {
    val config = themeManager.themeConfig.collectAsState(ThemeConfig()).value
    val colorScheme = themeManager.getColorScheme(config)
    
    MaterialTheme(
        colorScheme = colorScheme,
        content = content
    )
}
