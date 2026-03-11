import androidx.compose.ui.ExperimentalComposeUiApi
import androidx.compose.ui.window.CanvasBasedWindow
import com.russhwolf.settings.StorageSettings
import kotlinx.browser.localStorage
import org.jetbrains.skiko.wasm.onWasmReady

/**
 * Web (Wasm)平台入口
 */
@OptIn(ExperimentalComposeUiApi::class)
fun main() {
    onWasmReady {
        // 使用StorageSettings作为Web平台的存储实现（基于localStorage）
        val settings = StorageSettings(localStorage)
        
        CanvasBasedWindow(canvasElementId = "ComposeTarget") {
            App(settings)
        }
    }
}