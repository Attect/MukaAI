plugins {
    alias(libs.plugins.kotlin.multiplatform)
    alias(libs.plugins.kotlin.serialization)
}

kotlin {
    jvm()
    
    sourceSets {
        commonMain {
            dependencies {
                implementation(libs.ktor.client.core)
                implementation(libs.ktor.client.content.negotiation)
                implementation(libs.ktor.serialization.kotlinx.json)
                implementation(libs.ktor.client.cio)
                implementation(libs.kotlinx.coroutines.core)
                implementation(libs.kotlinx.serialization.json)
            }
        }
        
        jvmMain {
            dependencies {
                implementation(libs.ktor.client.cio.jvm)
            }
        }
    }
}

// 创建运行任务 - 使用配置收集依赖
tasks.register<JavaExec>("runTest") {
    group = "Execution"
    description = "Run LM Studio test"
    dependsOn("compileKotlinJvm")
    classpath = configurations.getByName("jvmRuntimeClasspath") + files(kotlin.jvm().compilations.getByName("main").output.classesDirs)
    mainClass.set("com.assistant.test.lmstudio.LMStudioTestKt")
    standardInput = System.`in`
}
