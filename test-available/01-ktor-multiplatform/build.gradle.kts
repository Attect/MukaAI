plugins {
    alias(libs.plugins.kotlin.multiplatform)
    alias(libs.plugins.kotlin.serialization)
}

kotlin {
    jvm()
    
    sourceSets {
        commonMain {
            dependencies {
                implementation(libs.ktor.server.core)
                implementation(libs.ktor.server.netty)
                implementation(libs.ktor.server.cors)
                implementation(libs.ktor.server.content.negotiation)
                implementation(libs.ktor.client.core)
                implementation(libs.ktor.client.content.negotiation)
                implementation(libs.ktor.serialization.kotlinx.json)
                implementation(libs.kotlinx.coroutines.core)
                implementation(libs.kotlinx.serialization.json)
            }
        }
        
        jvmMain {
            dependencies {
                implementation(libs.ktor.server.netty.jvm)
                implementation(libs.ktor.client.cio.jvm)
            }
        }
    }
}

// 创建运行任务
tasks.register<JavaExec>("runTest") {
    group = "Execution"
    description = "Run Ktor multiplatform test"
    dependsOn("compileKotlinJvm")
    classpath = configurations.getByName("jvmRuntimeClasspath") + files(kotlin.jvm().compilations.getByName("main").output.classesDirs)
    mainClass.set("KtorTestKt")
    standardInput = System.`in`
}
