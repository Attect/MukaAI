plugins {
    alias(libs.plugins.kotlin.multiplatform)
    alias(libs.plugins.kotlin.serialization)
}

kotlin {
    jvm()
    
    sourceSets {
        commonMain {
            dependencies {
                implementation(libs.kotlinx.serialization.json)
                implementation(libs.kotlinx.coroutines.core)
            }
        }
        
        jvmMain {
            dependencies {
                // JVM 特定的图像处理依赖可以在这里添加
            }
        }
    }
}

// 创建运行任务
tasks.register<JavaExec>("runTest") {
    group = "Execution"
    description = "Run multimodal session test"
    dependsOn("compileKotlinJvm")
    classpath = configurations.getByName("jvmRuntimeClasspath") + files(kotlin.jvm().compilations.getByName("main").output.classesDirs)
    mainClass.set("MultimodalSessionTestKt")
    standardInput = System.`in`
}
