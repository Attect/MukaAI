plugins {
    kotlin("jvm") version "2.3.10"
    application
}

application {
    mainClass.set("app.muka.ai.config.KotlinDslConfigLoaderKt")
}

repositories {
    mavenCentral()
}

dependencies {
    // Kotlin Scripting 支持（用于加载 .conf.kts 文件）
    implementation("org.jetbrains.kotlin:kotlin-scripting-jvm:2.3.10")
    implementation("org.jetbrains.kotlin:kotlin-scripting-common:2.3.10")
    implementation("org.jetbrains.kotlin:kotlin-scripting-jvm-host:2.3.10")
    
    // 命令行参数解析
    implementation("com.github.ajalt.clikt:clikt:4.4.0")
    
    // 测试依赖
    testImplementation("org.jetbrains.kotlin:kotlin-test")
}

kotlin {
    jvmToolchain(17)
}

tasks.test {
    useJUnitPlatform()
}