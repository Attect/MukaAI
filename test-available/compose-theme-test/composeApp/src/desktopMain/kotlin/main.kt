import androidx.compose.ui.window.Window
import androidx.compose.ui.window.application
import com.russhwolf.settings.PreferencesSettings
import java.util.prefs.Preferences

/**
 * Desktop平台入口
 */
fun main() = application {
    // 使用PreferencesSettings作为Desktop平台的存储实现
    val preferences = Preferences.userRoot().node("com.example.theme.test")
    val settings = PreferencesSettings(preferences)
    
    Window(
        onCloseRequest = ::exitApplication,
        title = "Compose Theme Test - Desktop"
    ) {
        App(settings)
    }
}