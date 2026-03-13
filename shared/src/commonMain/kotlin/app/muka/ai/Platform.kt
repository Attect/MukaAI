package app.muka.ai

interface Platform {
    val name: String
}

expect fun getPlatform(): Platform