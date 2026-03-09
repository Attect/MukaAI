plugins {
    alias(libs.plugins.kotlin.multiplatform)
}

kotlin {
    jvm()
    
    sourceSets {
        commonMain {
            dependencies {
                implementation(libs.kotlinx.coroutines.core)
                implementation(libs.kotlinx.io.core)
            }
        }
    }
}

// 创建运行任务
tasks.register<JavaExec>("runTest") {
    group = "Execution"
    description = "Run shell execution test"
    dependsOn("compileKotlinJvm")
    classpath = configurations.getByName("jvmRuntimeClasspath") + files(kotlin.jvm().compilations.getByName("main").output.classesDirs)
    mainClass.set("ShellExecutionTestKt")
    standardInput = System.`in`
}
