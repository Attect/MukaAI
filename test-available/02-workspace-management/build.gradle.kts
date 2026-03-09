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
                implementation(libs.kotlinx.io.core)
            }
        }
    }
}

// 创建运行任务
tasks.register<JavaExec>("runTest") {
    group = "Execution"
    description = "Run workspace management test"
    dependsOn("compileKotlinJvm")
    classpath = configurations.getByName("jvmRuntimeClasspath") + files(kotlin.jvm().compilations.getByName("main").output.classesDirs)
    mainClass.set("WorkspaceManagerTestKt")
    standardInput = System.`in`
}
