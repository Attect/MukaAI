plugins {
    kotlin("jvm") version "2.3.10"
    application
}

group = "com.assistant.test"
version = "1.0.0"

repositories {
    mavenCentral()
}

dependencies {
    // Kotlin 协程
    implementation(libs.kotlinx.coroutines.core)
    implementation(libs.kotlinx.coroutines.jdk8)
    testImplementation(libs.kotlinx.coroutines.test)
    
    // Ktor (服务端)
    implementation(libs.ktor.server.core.jvm)
    implementation(libs.ktor.server.cio.jvm)
    implementation(libs.ktor.server.content.negotiation)
    implementation(libs.ktor.server.status.pages)
    implementation(libs.ktor.server.cors)
    
    // Ktor (客户端)
    implementation(libs.ktor.client.core.jvm)
    implementation(libs.ktor.client.cio.jvm)
    implementation(libs.ktor.client.content.negotiation)
    
    // 序列化
    implementation(libs.kotlinx.serialization.json)
    implementation(libs.kotlinx.serialization.core)
    
    // 配置
    implementation(libs.typesafe.config)
    
    // 日志
    implementation(libs.kotlin.logging)
    implementation(libs.slf4j.api)
    implementation(libs.logback.classic)
    
    // 时间处理
    implementation(libs.kotlinx.datetime)
    
    // 测试
    testImplementation(kotlin("test"))
}

kotlin {
    jvmToolchain(17)
}

tasks.test {
    useJUnitPlatform()
}

tasks.withType<Test> {
    testLogging {
        events("passed", "skipped", "failed")
        showStandardStreams = true
    }
}
