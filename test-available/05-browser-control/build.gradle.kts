plugins {
    alias(libs.plugins.kotlin.multiplatform)
}

kotlin {
    jvm()
    
    sourceSets {
        commonMain {
            dependencies {
                implementation(libs.kotlinx.coroutines.core)
            }
        }
    }
}

// 创建运行任务
tasks.register<JavaExec>("runTest") {
    group = "Execution"
    description = "Run browser control test"
    dependsOn("compileKotlinJvm")
    classpath = configurations.getByName("jvmRuntimeClasspath") + files(kotlin.jvm().compilations.getByName("main").output.classesDirs)
    mainClass.set("BrowserControlTestKt")
    standardInput = System.`in`
}
