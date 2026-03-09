# Ktor 集成方案指南

本文档整理了 Ktor 在全栈开发和 RPC 集成方面的方案，包括 Kotlin Multiplatform 全栈应用、Kotlin RPC、Ktor 客户端与服务端集成等。

## 文档来源

- **官方文档**: 
  - https://ktor.io/docs/full-stack-development-with-kotlin-multiplatform.html
  - https://ktor.io/docs/tutorial-first-steps-with-kotlin-rpc.html
- **版本**: Ktor 3.4.1+, Kotlin 2.3.10+
- **最后更新**: 2026 年 3 月

## Kotlin Multiplatform 全栈开发

### 项目架构

Ktor 全栈开发使用 Kotlin Multiplatform (KMP) 实现代码共享，架构如下:

```
project/
├── shared/              # 共享模块 (commonMain, androidMain, iosMain)
│   ├── src/
│   │   ├── commonMain/  # 公共代码 (数据模型、API 客户端、业务逻辑)
│   │   ├── androidMain/ # Android 特定实现
│   │   └── iosMain/     # iOS 特定实现
│   └── build.gradle.kts
├── composeApp/          # Compose Multiplatform 客户端
│   ├── src/
│   │   ├── androidMain/ # Android 应用
│   │   ├── desktopMain/ # Desktop 应用
│   │   └── iosMain/     # iOS 应用
│   └── build.gradle.kts
├── server/              # Ktor 服务端
│   └── src/
│       └── main/
│   └── build.gradle.kts
└── build.gradle.kts     # 根项目构建文件
```

### 版本目录配置

在 `gradle/libs.versions.toml` 中统一管理依赖版本:

```toml
[versions]
kotlin = "2.3.10"
ktor = "3.4.1"
kotlinx-serialization = "1.8.0"
kotlinx-coroutines = "1.10.0"
kotlinx-datetime = "0.6.2"
sqldelight = "2.1.0"
koin = "4.0.0"
compose-plugin = "1.8.0"
agp = "8.8.0"

[libraries]
# Ktor 客户端
ktor-client-core = { module = "io.ktor:ktor-client-core", version.ref = "ktor" }
ktor-client-content-negotiation = { module = "io.ktor:ktor-client-content-negotiation", version.ref = "ktor" }
ktor-serialization-kotlinx-json = { module = "io.ktor:ktor-serialization-kotlinx-json", version.ref = "ktor" }
ktor-client-android = { module = "io.ktor:ktor-client-android", version.ref = "ktor" }
ktor-client-darwin = { module = "io.ktor:ktor-client-darwin", version.ref = "ktor" }
ktor-client-okhttp = { module = "io.ktor:ktor-client-okhttp", version.ref = "ktor" }

# Ktor 服务端
ktor-server-core = { module = "io.ktor:ktor-server-core", version.ref = "ktor" }
ktor-server-netty = { module = "io.ktor:ktor-server-netty", version.ref = "ktor" }
ktor-server-content-negotiation = { module = "io.ktor:ktor-server-content-negotiation", version.ref = "ktor" }
ktor-server-cors = { module = "io.ktor:ktor-server-cors", version.ref = "ktor" }

# 序列化
kotlinx-serialization-json = { module = "org.jetbrains.kotlinx:kotlinx-serialization-json", version.ref = "kotlinx-serialization" }

# 协程
kotlinx-coroutines-core = { module = "org.jetbrains.kotlinx:kotlinx-coroutines-core", version.ref = "kotlinx-coroutines" }

# 日期时间
kotlinx-datetime = { module = "org.jetbrains.kotlinx:kotlinx-datetime", version.ref = "kotlinx-datetime" }

# SQLDelight
sqldelight-runtime = { module = "app.cash.sqldelight:runtime", version.ref = "sqldelight" }
sqldelight-android-driver = { module = "app.cash.sqldelight:android-driver", version.ref = "sqldelight" }
sqldelight-native-driver = { module = "app.cash.sqldelight:native-driver", version.ref = "sqldelight" }
sqldelight-coroutines = { module = "app.cash.sqldelight:coroutines-extensions", version.ref = "sqldelight" }

# Koin DI
koin-core = { module = "io.insert-koin:koin-core", version.ref = "koin" }
koin-android = { module = "io.insert-koin:koin-android", version.ref = "koin" }
koin-compose = { module = "io.insert-koin:koin-compose", version.ref = "koin" }

[plugins]
kotlin-multiplatform = { id = "org.jetbrains.kotlin.multiplatform", version.ref = "kotlin" }
kotlin-serialization = { id = "org.jetbrains.kotlin.plugin.serialization", version.ref = "kotlin" }
android-application = { id = "com.android.application", version.ref = "agp" }
android-library = { id = "com.android.library", version.ref = "agp" }
compose-compiler = { id = "org.jetbrains.kotlin.plugin.compose", version.ref = "kotlin" }
sqldelight = { id = "app.cash.sqldelight", version.ref = "sqldelight" }
```

### 共享模块配置

**shared/build.gradle.kts**:

```kotlin
plugins {
    alias(libs.plugins.kotlin.multiplatform)
    alias(libs.plugins.kotlin.serialization)
    alias(libs.plugins.sqldelight)
}

kotlin {
    androidTarget()
    
    iosX64()
    iosArm64()
    iosSimulatorArm64()
    
    sourceSets {
        commonMain.dependencies {
            implementation(libs.ktor.client.core)
            implementation(libs.ktor.client.content.negotiation)
            implementation(libs.ktor.serialization.kotlinx.json)
            implementation(libs.kotlinx.serialization.json)
            implementation(libs.kotlinx.coroutines.core)
            implementation(libs.kotlinx.datetime)
            implementation(libs.sqldelight.runtime)
            implementation(libs.sqldelight.coroutines)
            implementation(libs.koin.core)
        }
        
        androidMain.dependencies {
            implementation(libs.ktor.client.android)
            implementation(libs.sqldelight.android.driver)
            implementation(libs.koin.android)
        }
        
        iosMain.dependencies {
            implementation(libs.ktor.client.darwin)
            implementation(libs.sqldelight.native.driver)
        }
    }
}

android {
    namespace = "com.example.shared"
    compileSdk = 35
    defaultConfig {
        minSdk = 24
    }
}

sqldelight {
    databases {
        create("AppDatabase") {
            packageName.set("com.example.shared.database")
        }
    }
}
```

### 数据模型共享

在 `shared/src/commonMain/kotlin` 中定义可序列化的数据类:

```kotlin
package com.example.shared.model

import kotlinx.datetime.Instant
import kotlinx.serialization.SerialName
import kotlinx.serialization.Serializable

@Serializable
data class User(
    @SerialName("id")
    val id: Long,
    
    @SerialName("name")
    val name: String,
    
    @SerialName("email")
    val email: String,
    
    @SerialName("createdAt")
    val createdAt: Instant
)

@Serializable
data class CreateUserRequest(
    @SerialName("name")
    val name: String,
    
    @SerialName("email")
    val email: String
)

@Serializable
data class ApiResponse<T>(
    @SerialName("success")
    val success: Boolean,
    
    @SerialName("data")
    val data: T?,
    
    @SerialName("message")
    val message: String? = null
)
```

### Ktor 服务端实现

**server/src/main/kotlin/com/example/server/Application.kt**:

```kotlin
package com.example.server

import com.example.server.model.User
import com.example.server.plugins.configureContentNegotiation
import com.example.server.plugins.configureCORS
import com.example.server.plugins.configureRouting
import io.ktor.server.application.*
import io.ktor.server.engine.*
import io.ktor.server.netty.*

fun main() {
    embeddedServer(Netty, port = 8080, host = "0.0.0.0") {
        configureContentNegotiation()
        configureCORS()
        configureRouting()
    }.start(wait = true)
}

fun Application.module() {
    configureContentNegotiation()
    configureCORS()
    configureRouting()
}
```

**server/src/main/kotlin/com/example/server/plugins/ContentNegotiation.kt**:

```kotlin
package com.example.server.plugins

import io.ktor.serialization.kotlinx.json.*
import io.ktor.server.plugins.contentnegotiation.*
import io.ktor.server.application.*
import kotlinx.serialization.json.Json

fun Application.configureContentNegotiation() {
    install(ContentNegotiation) {
        json(Json {
            prettyPrint = true
            isLenient = true
            ignoreUnknownKeys = true
            encodeDefaults = true
        })
    }
}
```

**server/src/main/kotlin/com/example/server/plugins/CORS.kt**:

```kotlin
package com.example.server.plugins

import io.ktor.http.*
import io.ktor.server.plugins.cors.routing.*
import io.ktor.server.application.*

fun Application.configureCORS() {
    install(CORS) {
        allowMethod(HttpMethod.Get)
        allowMethod(HttpMethod.Post)
        allowMethod(HttpMethod.Put)
        allowMethod(HttpMethod.Delete)
        allowMethod(HttpMethod.Options)
        
        allowHeader(HttpHeaders.ContentType)
        allowHeader(HttpHeaders.Authorization)
        
        allowCredentials = true
        
        // 开发环境允许所有来源
        anyHost()
        
        // 生产环境指定来源
        // host("https://yourdomain.com")
    }
}
```

**server/src/main/kotlin/com/example/server/routes/UserRoutes.kt**:

```kotlin
package com.example.server.routes

import com.example.server.model.User
import com.example.server.model.CreateUserRequest
import io.ktor.http.*
import io.ktor.server.application.*
import io.ktor.server.request.*
import io.ktor.server.response.*
import io.ktor.server.routing.*

fun Routing.userRoutes(users: MutableList<User> = mutableListOf()) {
    route("/api/users") {
        get {
            call.respond(users)
        }
        
        get("/{id}") {
            val id = call.parameters["id"]?.toLongOrNull()
            val user = users.find { it.id == id }
            
            if (user != null) {
                call.respond(user)
            } else {
                call.respond(HttpStatusCode.NotFound, "User not found")
            }
        }
        
        post {
            val request = call.receive<CreateUserRequest>()
            
            val newUser = User(
                id = users.maxOfOrNull { it.id }?.plus(1) ?: 1L,
                name = request.name,
                email = request.email,
                createdAt = kotlinx.datetime.Clock.System.now()
            )
            
            users.add(newUser)
            call.respond(HttpStatusCode.Created, newUser)
        }
        
        put("/{id}") {
            val id = call.parameters["id"]?.toLongOrNull()
            val request = call.receive<CreateUserRequest>()
            
            val index = users.indexOfFirst { it.id == id }
            if (index != -1) {
                val updatedUser = users[index].copy(
                    name = request.name,
                    email = request.email
                )
                users[index] = updatedUser
                call.respond(updatedUser)
            } else {
                call.respond(HttpStatusCode.NotFound, "User not found")
            }
        }
        
        delete("/{id}") {
            val id = call.parameters["id"]?.toLongOrNull()
            val removed = users.removeAll { it.id == id }
            
            if (removed) {
                call.respond(HttpStatusCode.NoContent)
            } else {
                call.respond(HttpStatusCode.NotFound, "User not found")
            }
        }
    }
}
```

### Ktor 客户端实现 (共享模块)

**shared/src/commonMain/kotlin/com/example/shared/api/ApiClient.kt**:

```kotlin
package com.example.shared.api

import com.example.shared.model.User
import com.example.shared.model.CreateUserRequest
import io.ktor.client.*
import io.ktor.client.call.*
import io.ktor.client.plugins.contentnegotiation.*
import io.ktor.client.request.*
import io.ktor.serialization.kotlinx.json.*
import kotlinx.serialization.json.Json

class ApiClient(private val baseUrl: String) {
    private val client = HttpClient {
        install(ContentNegotiation) {
            json(Json {
                prettyPrint = true
                isLenient = true
                ignoreUnknownKeys = true
            })
        }
    }
    
    suspend fun getUsers(): List<User> {
        return client.get("$baseUrl/api/users").body()
    }
    
    suspend fun getUser(id: Long): User? {
        return try {
            client.get("$baseUrl/api/users/$id").body()
        } catch (e: Exception) {
            null
        }
    }
    
    suspend fun createUser(request: CreateUserRequest): User {
        return client.post("$baseUrl/api/users") {
            contentType(ContentType.Application.Json)
            setBody(request)
        }.body()
    }
    
    suspend fun updateUser(id: Long, request: CreateUserRequest): User {
        return client.put("$baseUrl/api/users/$id") {
            contentType(ContentType.Application.Json)
            setBody(request)
        }.body()
    }
    
    suspend fun deleteUser(id: Long): Boolean {
        return try {
            client.delete("$baseUrl/api/users/$id")
            true
        } catch (e: Exception) {
            false
        }
    }
}
```

**shared/src/commonMain/kotlin/com/example/shared/SpaceXSDK.kt**:

```kotlin
package com.example.shared

import com.example.shared.api.ApiClient
import com.example.shared.cache.Database
import com.example.shared.cache.DatabaseDriverFactory
import com.example.shared.model.User
import com.example.shared.model.CreateUserRequest

class SpaceXSDK(databaseDriverFactory: DatabaseDriverFactory) {
    private val database = Database(databaseDriverFactory)
    private val apiClient = ApiClient("http://localhost:8080")
    
    @Throws(Exception::class)
    suspend fun getUsers(forceReload: Boolean = false): List<User> {
        val cachedUsers = database.getAllUsers()
        
        return if (cachedUsers.isNotEmpty() && !forceReload) {
            cachedUsers
        } else {
            try {
                val users = apiClient.getUsers()
                database.clearAndCreateUsers(users)
                users
            } catch (e: Exception) {
                if (cachedUsers.isEmpty()) {
                    throw e
                }
                cachedUsers
            }
        }
    }
    
    @Throws(Exception::class)
    suspend fun createUser(request: CreateUserRequest): User {
        val user = apiClient.createUser(request)
        database.insertUser(user)
        return user
    }
}
```

### 数据库集成 (SQLDelight)

**shared/src/commonMain/sqldelight/com/example/shared/database/AppDatabase.sq**:

```sql
import com.example.shared.model.User;

CREATE TABLE User (
    id INTEGER NOT NULL,
    name TEXT NOT NULL,
    email TEXT NOT NULL,
    createdAt TEXT NOT NULL,
    PRIMARY KEY (id)
);

selectAllUsers:
SELECT * FROM User;

selectUserById:
SELECT * FROM User WHERE id = ?;

insertUser:
INSERT INTO User (id, name, email, createdAt)
VALUES (?, ?, ?, ?);

deleteAllUsers:
DELETE FROM User;

deleteUserById:
DELETE FROM User WHERE id = ?;
```

**shared/src/commonMain/kotlin/com/example/shared/cache/Database.kt**:

```kotlin
package com.example.shared.cache

import com.example.shared.database.AppDatabase
import com.example.shared.database.User
import com.example.shared.model.User as ModelUser

internal class Database(databaseDriverFactory: DatabaseDriverFactory) {
    private val database = AppDatabase(databaseDriverFactory.createDriver())
    private val queries = database.appDatabaseQueries
    
    internal fun getAllUsers(): List<ModelUser> {
        return queries.selectAllUsers().executeAsList().map { it.toModelUser() }
    }
    
    internal fun getUserById(id: Long): ModelUser? {
        return queries.selectUserById(id).executeAsOneOrNull()?.toModelUser()
    }
    
    internal fun insertUser(user: ModelUser) {
        queries.insertUser(
            id = user.id,
            name = user.name,
            email = user.email,
            createdAt = user.createdAt.toString()
        )
    }
    
    internal fun clearAndCreateUsers(users: List<ModelUser>) {
        queries.transaction {
            queries.deleteAllUsers()
            users.forEach { user ->
                insertUser(user)
            }
        }
    }
    
    internal fun deleteUser(id: Long) {
        queries.deleteUserById(id)
    }
}

// 扩展函数：SQLDelight User -> Model User
private fun User.toModelUser(): ModelUser {
    return ModelUser(
        id = id,
        name = name,
        email = email,
        createdAt = kotlinx.datetime.Instant.parse(createdAt)
    )
}
```

**shared/src/commonMain/kotlin/com/example/shared/cache/DatabaseDriverFactory.kt**:

```kotlin
package com.example.shared.cache

import app.cash.sqldelight.db.SqlDriver

interface DatabaseDriverFactory {
    fun createDriver(): SqlDriver
}
```

**shared/src/androidMain/kotlin/com/example/shared/cache/AndroidDatabaseDriverFactory.kt**:

```kotlin
package com.example.shared.cache

import android.content.Context
import app.cash.sqldelight.db.SqlDriver
import app.cash.sqldelight.driver.android.AndroidSqliteDriver
import com.example.shared.database.AppDatabase

class AndroidDatabaseDriverFactory(
    private val context: Context
) : DatabaseDriverFactory {
    override fun createDriver(): SqlDriver {
        return AndroidSqliteDriver(
            schema = AppDatabase.Schema,
            context = context,
            name = "app.db"
        )
    }
}
```

**shared/src/iosMain/kotlin/com/example/shared/cache/IOSDatabaseDriverFactory.kt**:

```kotlin
package com.example.shared.cache

import app.cash.sqldelight.db.SqlDriver
import app.cash.sqldelight.driver.native.NativeSqliteDriver
import com.example.shared.database.AppDatabase

class IOSDatabaseDriverFactory : DatabaseDriverFactory {
    override fun createDriver(): SqlDriver {
        return NativeSqliteDriver(
            schema = AppDatabase.Schema,
            name = "app.db"
        )
    }
}
```

### 依赖注入 (Koin)

**shared/src/commonMain/kotlin/com/example/shared/di/SharedModule.kt**:

```kotlin
package com.example.shared.di

import com.example.shared.SpaceXSDK
import com.example.shared.cache.DatabaseDriverFactory
import org.koin.core.module.dsl.singleOf
import org.koin.dsl.bind
import org.koin.dsl.module

val sharedModule = module {
    single { SpaceXSDK(get()) }
    singleOf(::DatabaseDriverFactory) bind DatabaseDriverFactory::class
}
```

**shared/src/androidMain/kotlin/com/example/shared/di/AndroidModule.kt**:

```kotlin
package com.example.shared.di

import android.content.Context
import com.example.shared.cache.AndroidDatabaseDriverFactory
import com.example.shared.cache.DatabaseDriverFactory
import org.koin.android.ext.koin.androidContext
import org.koin.core.module.dsl.singleOf
import org.koin.dsl.bind
import org.koin.dsl.module

val androidModule = module {
    single { AndroidDatabaseDriverFactory(androidContext()) } bind DatabaseDriverFactory::class
}
```

**shared/src/iosMain/kotlin/com/example/shared/di/IOSModule.kt**:

```kotlin
package com.example.shared.di

import com.example.shared.cache.DatabaseDriverFactory
import com.example.shared.cache.IOSDatabaseDriverFactory
import org.koin.core.module.dsl.singleOf
import org.koin.dsl.bind
import org.koin.dsl.module

val iosModule = module {
    singleOf(::IOSDatabaseDriverFactory) bind DatabaseDriverFactory::class
}
```

**Android Application 类**:

```kotlin
package com.example.android

import android.app.Application
import com.example.shared.di.androidModule
import com.example.shared.di.sharedModule
import org.koin.android.ext.koin.androidContext
import org.koin.android.ext.koin.androidLogger
import org.koin.core.context.startKoin
import org.koin.core.logger.Level

class MainApplication : Application() {
    override fun onCreate() {
        super.onCreate()
        
        startKoin {
            androidLogger(Level.DEBUG)
            androidContext(this@MainApplication)
            modules(
                sharedModule,
                androidModule
            )
        }
    }
}
```

**iOS KoinHelper 类**:

```kotlin
package com.example.shared

import com.example.shared.SpaceXSDK
import com.example.shared.di.iosModule
import com.example.shared.di.sharedModule
import org.koin.core.context.startKoin
import org.koin.core.component.KoinComponent
import org.koin.core.component.inject

class KoinHelper : KoinComponent {
    private val sdk: SpaceXSDK by inject()
    
    suspend fun getUsers(forceReload: Boolean) = sdk.getUsers(forceReload)
    
    fun initKoin() {
        startKoin {
            modules(
                sharedModule,
                iosModule
            )
        }
    }
}

// 导出给 Swift 使用
fun initKoin() {
    KoinHelper().initKoin()
}
```

### Compose Multiplatform 客户端

**composeApp/build.gradle.kts**:

```kotlin
import org.jetbrains.compose.desktop.application.dsl.TargetFormat

plugins {
    alias(libs.plugins.kotlin.multiplatform)
    alias(libs.plugins.compose.compiler)
    alias(libs.plugins.android.application)
}

kotlin {
    androidTarget()
    
    jvm("desktop")
    
    iosX64()
    iosArm64()
    iosSimulatorArm64()
    
    sourceSets {
        val desktopMain by getting
        
        commonMain.dependencies {
            implementation(project(":shared"))
            implementation(compose.runtime)
            implementation(compose.foundation)
            implementation(compose.material3)
            implementation(compose.ui)
            implementation(compose.components.resources)
            implementation(libs.koin.compose)
        }
        
        androidMain.dependencies {
            implementation(compose.preview)
            implementation(libs.androidx.activity.compose)
            implementation(libs.androidx.lifecycle.viewmodel)
        }
        
        desktopMain.dependencies {
            implementation(compose.desktop.currentOs)
        }
        
        iosMain.dependencies {
        }
    }
}

android {
    namespace = "com.example.composeapp"
    compileSdk = 35
    
    defaultConfig {
        applicationId = "com.example.composeapp"
        minSdk = 24
        targetSdk = 35
        versionCode = 1
        versionName = "1.0"
    }
    
    buildTypes {
        release {
            isMinifyEnabled = false
        }
    }
    
    compileOptions {
        sourceCompatibility = JavaVersion.VERSION_17
        targetCompatibility = JavaVersion.VERSION_17
    }
}

compose.desktop {
    application {
        mainClass = "com.example.desktop.MainKt"
        
        nativeDistributions {
            targetFormats(TargetFormat.Dmg, TargetFormat.Msi, TargetFormat.Deb)
            packageName = "composeApp"
            packageVersion = "1.0.0"
        }
    }
}
```

**Android 视图模型**:

```kotlin
package com.example.android

import androidx.lifecycle.ViewModel
import androidx.lifecycle.viewModelScope
import com.example.shared.SpaceXSDK
import com.example.shared.model.User
import kotlinx.coroutines.flow.MutableStateFlow
import kotlinx.coroutines.flow.StateFlow
import kotlinx.coroutines.flow.asStateFlow
import kotlinx.coroutines.launch

class UserViewModel(private val sdk: SpaceXSDK) : ViewModel() {
    private val _uiState = MutableStateFlow(UserUiState())
    val uiState: StateFlow<UserUiState> = _uiState.asStateFlow()
    
    init {
        loadUsers()
    }
    
    fun loadUsers() {
        viewModelScope.launch {
            _uiState.value = UserUiState(isLoading = true)
            try {
                val users = sdk.getUsers(forceReload = true)
                _uiState.value = UserUiState(users = users)
            } catch (e: Exception) {
                _uiState.value = UserUiState(error = e.message)
            }
        }
    }
    
    fun createUser(name: String, email: String) {
        viewModelScope.launch {
            try {
                sdk.createUser(CreateUserRequest(name, email))
                loadUsers()
            } catch (e: Exception) {
                _uiState.value = _uiState.value.copy(error = e.message)
            }
        }
    }
}

data class UserUiState(
    val isLoading: Boolean = false,
    val users: List<User> = emptyList(),
    val error: String? = null
)
```

**Android Composable UI**:

```kotlin
package com.example.android

import androidx.compose.foundation.layout.*
import androidx.compose.foundation.lazy.LazyColumn
import androidx.compose.foundation.lazy.items
import androidx.compose.material3.*
import androidx.compose.runtime.*
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.unit.dp
import com.example.shared.model.User
import org.koin.androidx.compose.koinViewModel

@Composable
fun App() {
    val viewModel: UserViewModel = koinViewModel()
    val uiState by viewModel.uiState.collectAsState()
    
    MaterialTheme {
        Surface(
            modifier = Modifier.fillMaxSize(),
            color = MaterialTheme.colorScheme.background
        ) {
            UserScreen(
                uiState = uiState,
                onRefresh = { viewModel.loadUsers() },
                onCreateUser = { name, email ->
                    viewModel.createUser(name, email)
                }
            )
        }
    }
}

@OptIn(ExperimentalMaterial3Api::class)
@Composable
fun UserScreen(
    uiState: UserUiState,
    onRefresh: () -> Unit,
    onCreateUser: (String, String) -> Unit
) {
    val pullToRefreshState = rememberPullToRefreshState()
    
    if (pullToRefreshState.isRefreshing) {
        LaunchedEffect(Unit) {
            onRefresh()
            pullToRefreshState.endRefresh()
        }
    }
    
    Box(
        modifier = Modifier
            .fillMaxSize()
            .pullToRefresh(
                isRefreshing = uiState.isLoading,
                state = pullToRefreshState,
                onRefresh = onRefresh
            )
    ) {
        Column(
            modifier = Modifier
                .fillMaxSize()
                .padding(16.dp)
        ) {
            if (uiState.error != null) {
                Text(
                    text = "错误：${uiState.error}",
                    color = MaterialTheme.colorScheme.error
                )
            }
            
            LazyColumn(
                modifier = Modifier.weight(1f),
                verticalArrangement = Arrangement.spacedBy(8.dp)
            ) {
                items(uiState.users) { user ->
                    UserItem(user)
                }
            }
            
            CreateUserForm(onCreateUser = onCreateUser)
        }
        
        if (uiState.isLoading) {
            CircularProgressIndicator(
                modifier = Modifier.align(Alignment.Center)
            )
        }
    }
}

@Composable
fun UserItem(user: User) {
    Card(
        modifier = Modifier.fillMaxWidth(),
        elevation = CardDefaults.cardElevation(defaultElevation = 2.dp)
    ) {
        Column(
            modifier = Modifier.padding(16.dp)
        ) {
            Text(
                text = user.name,
                style = MaterialTheme.typography.titleMedium
            )
            Text(
                text = user.email,
                style = MaterialTheme.typography.bodyMedium,
                color = MaterialTheme.colorScheme.onSurfaceVariant
            )
        }
    }
}

@Composable
fun CreateUserForm(onCreateUser: (String, String) -> Unit) {
    var name by remember { mutableStateOf("") }
    var email by remember { mutableStateOf("") }
    
    Column(
        verticalArrangement = Arrangement.spacedBy(8.dp)
    ) {
        OutlinedTextField(
            value = name,
            onValueChange = { name = it },
            label = { Text("姓名") },
            modifier = Modifier.fillMaxWidth(),
            singleLine = true
        )
        
        OutlinedTextField(
            value = email,
            onValueChange = { email = it },
            label = { Text("邮箱") },
            modifier = Modifier.fillMaxWidth(),
            singleLine = true
        )
        
        Button(
            onClick = {
                if (name.isNotBlank() && email.isNotBlank()) {
                    onCreateUser(name, email)
                    name = ""
                    email = ""
                }
            },
            modifier = Modifier.fillMaxWidth()
        ) {
            Text("添加用户")
        }
    }
}
```

**iOS SwiftUI 视图**:

```swift
import SwiftUI
import Shared

struct ContentView: View {
    @StateObject private var viewModel = UserViewModel()
    
    var body: some View {
        NavigationView {
            VStack {
                if let error = viewModel.error {
                    Text("错误：\(error)")
                        .foregroundColor(.red)
                        .padding()
                }
                
                List(viewModel.users) { user in
                    UserRow(user: user)
                }
                
                CreateUserForm(onCreateUser: viewModel.createUser)
            }
            .navigationTitle("用户列表")
            .refreshable {
                await viewModel.loadUsers()
            }
            .task {
                await viewModel.loadUsers()
            }
        }
    }
}

struct UserRow: View {
    let user: User
    
    var body: some View {
        VStack(alignment: .leading, spacing: 4) {
            Text(user.name)
                .font(.headline)
            Text(user.email)
                .font(.subheadline)
                .foregroundColor(.gray)
        }
    }
}

struct CreateUserForm: View {
    @State private var name = ""
    @State private var email = ""
    let onCreateUser: (String, String) -> Void
    
    var body: some View {
        VStack(spacing: 8) {
            TextField("姓名", text: $name)
                .textFieldStyle(RoundedBorderTextFieldStyle())
            
            TextField("邮箱", text: $email)
                .textFieldStyle(RoundedBorderTextFieldStyle())
            
            Button("添加用户") {
                onCreateUser(name, email)
                name = ""
                email = ""
            }
            .buttonStyle(.borderedProminent)
        }
        .padding()
    }
}

class UserViewModel: ObservableObject {
    @Published var users: [User] = []
    @Published var error: String?
    @Published var isLoading = false
    
    private let sdk: SpaceXSDK
    
    init() {
        self.sdk = SpaceXSDK(databaseDriverFactory: IOSDatabaseDriverFactory())
    }
    
    @MainActor
    func loadUsers() async {
        isLoading = true
        error = nil
        
        do {
            users = try await sdk.getUsers(forceReload: true)
        } catch {
            self.error = error.localizedDescription
        }
        
        isLoading = false
    }
    
    @MainActor
    func createUser(name: String, email: String) {
        Task {
            do {
                let request = CreateUserRequest(name: name, email: email)
                _ = try await sdk.createUser(request: request)
                await loadUsers()
            } catch {
                self.error = error.localizedDescription
            }
        }
    }
}
```

## Kotlin RPC 集成

### RPC 概述

Kotlin RPC (Remote Procedure Call) 是 Kotlin 生态系统中的新成员，基于 kotlinx.rpc 库构建。它允许你使用纯 Kotlin 语言结构进行跨网络的过程调用，是 REST 和 gRPC 的替代方案。

**核心特性**:
- 类型安全的 RPC 调用
- 使用 Kotlin 协程进行异步通信
- 支持多平台 (JVM、JS、Native)
- 传输机制不可知 (支持 WebSocket、HTTP、gRPC 等)
- 与 Kotlin 序列化无缝集成

### 添加 RPC 依赖

**根项目 build.gradle.kts**:

```kotlin
plugins {
    kotlin("multiplatform") version "2.3.10" apply false
    kotlin("plugin.serialization") version "2.3.10" apply false
    id("org.jetbrains.kotlinx.rpc.plugin") version "0.3.1" apply false
}
```

**shared/build.gradle.kts**:

```kotlin
plugins {
    alias(libs.plugins.kotlin.multiplatform)
    alias(libs.plugins.kotlin.serialization)
    id("org.jetbrains.kotlinx.rpc.plugin") version "0.3.1"
}

kotlin {
    jvm()
    js(IR) { browser() }
    iosX64()
    iosArm64()
    iosSimulatorArm64()
    
    sourceSets {
        commonMain.dependencies {
            implementation("org.jetbrains.kotlinx:kotlinx-rpc:0.3.1")
            implementation("org.jetbrains.kotlinx:kotlinx-rpc-serialization-json:0.3.1")
            implementation(libs.kotlinx.serialization.json)
        }
        
        jvmMain.dependencies {
            implementation("org.jetbrains.kotlinx:kotlinx-rpc-krpc-server:0.3.1")
            implementation("io.ktor:ktor-server-netty:$ktor_version")
        }
        
        jsMain.dependencies {
            implementation("org.jetbrains.kotlinx:kotlinx-rpc-krpc-client:0.3.1")
            implementation("io.ktor:ktor-client-core:$ktor_version")
        }
        
        iosMain.dependencies {
            implementation("org.jetbrains.kotlinx:kotlinx-rpc-krpc-client:0.3.1")
            implementation("io.ktor:ktor-client-darwin:$ktor_version")
        }
    }
}
```

### 定义 RPC 服务接口

**shared/src/commonMain/kotlin/com/example/shared/rpc/NewsService.kt**:

```kotlin
package com.example.shared.rpc

import kotlinx.serialization.Serializable

@Serializable
data class NewsArticle(
    val id: Long,
    val title: String,
    val content: String,
    val author: String,
    val publishedAt: String,
    val category: String
)

@Serializable
data class CreateArticleRequest(
    val title: String,
    val content: String,
    val author: String,
    val category: String
)

interface NewsService {
    suspend fun getArticles(category: String?): List<NewsArticle>
    suspend fun getArticleById(id: Long): NewsArticle?
    suspend fun createArticle(request: CreateArticleRequest): NewsArticle
    suspend fun deleteArticle(id: Long): Boolean
}
```

### RPC 服务端实现

**server/src/main/kotlin/com/example/server/rpc/NewsServiceImpl.kt**:

```kotlin
package com.example.server.rpc

import com.example.shared.rpc.NewsService
import com.example.shared.rpc.NewsArticle
import com.example.shared.rpc.CreateArticleRequest

class NewsServiceImpl : NewsService {
    private val articles = mutableListOf<NewsArticle>(
        NewsArticle(
            id = 1,
            title = "Kotlin 2.3.10 发布",
            content = "JetBrains 发布了 Kotlin 2.3.10...",
            author = "张三",
            publishedAt = "2025-12-01T10:00:00Z",
            category = "技术"
        )
    )
    
    override suspend fun getArticles(category: String?): List<NewsArticle> {
        return if (category != null) {
            articles.filter { it.category == category }
        } else {
            articles
        }
    }
    
    override suspend fun getArticleById(id: Long): NewsArticle? {
        return articles.find { it.id == id }
    }
    
    override suspend fun createArticle(request: CreateArticleRequest): NewsArticle {
        val newArticle = NewsArticle(
            id = articles.maxOfOrNull { it.id }?.plus(1) ?: 1L,
            title = request.title,
            content = request.content,
            author = request.author,
            publishedAt = java.time.Instant.now().toString(),
            category = request.category
        )
        articles.add(newArticle)
        return newArticle
    }
    
    override suspend fun deleteArticle(id: Long): Boolean {
        return articles.removeAll { it.id == id }
    }
}
```

**server/src/main/kotlin/com/example/server/Application.kt** (带 RPC):

```kotlin
package com.example.server

import com.example.server.rpc.NewsServiceImpl
import io.ktor.http.*
import io.ktor.serialization.kotlinx.*
import io.ktor.serialization.kotlinx.json.*
import io.ktor.server.application.*
import io.ktor.server.engine.*
import io.ktor.server.netty.*
import io.ktor.server.plugins.contentnegotiation.*
import io.ktor.server.plugins.cors.routing.*
import io.ktor.server.routing.*
import io.ktor.server.websocket.*
import kotlinx.coroutines.*
import kotlinx.rpc.krpc.KRpc
import kotlinx.rpc.krpc.krpc
import kotlinx.serialization.json.Json

fun main() {
    embeddedServer(Netty, port = 8080, host = "0.0.0.0") {
        configurePlugins()
        configureRPC()
    }.start(wait = true)
}

fun Application.configurePlugins() {
    install(CORS) {
        allowMethod(HttpMethod.Get)
        allowMethod(HttpMethod.Post)
        allowMethod(HttpMethod.Put)
        allowMethod(HttpMethod.Delete)
        allowMethod(HttpMethod.Options)
        allowHeader(HttpHeaders.ContentType)
        allowHeader(HttpHeaders.Authorization)
        allowCredentials = true
        anyHost()
    }
    
    install(ContentNegotiation) {
        json(Json {
            prettyPrint = true
            isLenient = true
            ignoreUnknownKeys = true
        })
    }
    
    install(WebSockets) {
        contentConverter = KotlinxWebsocketSerde(Json {
            ignoreUnknownKeys = true
        })
        pingPeriod = 10000
    }
}

fun Application.configureRPC() {
    val newsService = NewsServiceImpl()
    
    install(KRpc) {
        endpoint("/rpc") {
            service<NewsService> {
                getArticles = newsService::getArticles
                getArticleById = newsService::getArticleById
                createArticle = newsService::createArticle
                deleteArticle = newsService::deleteArticle
            }
        }
    }
    
    // 保留 REST API 作为备选
    routing {
        route("/api/news") {
            get {
                val category = call.request.queryParameters["category"]
                call.respond(newsService.getArticles(category))
            }
        }
    }
}
```

### RPC 客户端实现

**shared/src/commonMain/kotlin/com/example/shared/rpc/RpcClient.kt**:

```kotlin
package com.example.shared.rpc

import kotlinx.rpc.krpc.krpc
import kotlinx.rpc.withService

class RpcClient(private val baseUrl: String) {
    private val rpcClient = krpc {
        url("$baseUrl/rpc")
        rpcConfig {
            serialization {
                json()
            }
        }
    }
    
    private val newsService = rpcClient.withService<NewsService>()
    
    suspend fun getArticles(category: String? = null): List<NewsArticle> {
        return newsService.getArticles(category)
    }
    
    suspend fun getArticleById(id: Long): NewsArticle? {
        return newsService.getArticleById(id)
    }
    
    suspend fun createArticle(request: CreateArticleRequest): NewsArticle {
        return newsService.createArticle(request)
    }
    
    suspend fun deleteArticle(id: Long): Boolean {
        return newsService.deleteArticle(id)
    }
}
```

**shared/src/commonMain/kotlin/com/example/shared/NewsSDK.kt**:

```kotlin
package com.example.shared

import com.example.shared.cache.Database
import com.example.shared.cache.DatabaseDriverFactory
import com.example.shared.rpc.NewsArticle
import com.example.shared.rpc.CreateArticleRequest
import com.example.shared.rpc.RpcClient

class NewsSDK(databaseDriverFactory: DatabaseDriverFactory, rpcUrl: String) {
    private val database = Database(databaseDriverFactory)
    private val rpcClient = RpcClient(rpcUrl)
    
    @Throws(Exception::class)
    suspend fun getArticles(
        category: String? = null,
        forceReload: Boolean = false
    ): List<NewsArticle> {
        val cachedArticles = database.getAllArticles()
        
        return if (cachedArticles.isNotEmpty() && !forceReload) {
            cachedArticles
        } else {
            try {
                val articles = rpcClient.getArticles(category)
                database.clearAndCreateArticles(articles)
                articles
            } catch (e: Exception) {
                if (cachedArticles.isEmpty()) {
                    throw e
                }
                cachedArticles
            }
        }
    }
    
    @Throws(Exception::class)
    suspend fun getArticleById(id: Long): NewsArticle? {
        return rpcClient.getArticleById(id)
    }
    
    @Throws(Exception::class)
    suspend fun createArticle(request: CreateArticleRequest): NewsArticle {
        val article = rpcClient.createArticle(request)
        database.insertArticle(article)
        return article
    }
    
    @Throws(Exception::class)
    suspend fun deleteArticle(id: Long) {
        rpcClient.deleteArticle(id)
        database.deleteArticle(id)
    }
}
```

### RPC vs REST 对比

| 特性 | RPC | REST |
|------|-----|------|
| 类型安全 | ✅ 完全类型安全 | ⚠️ 需要手动定义数据类 |
| 代码生成 | ✅ 自动生成 | ❌ 需要 Swagger/OpenAPI |
| 学习曲线 | ✅ 低 (纯 Kotlin) | ⚠️ 中 (HTTP 语义) |
| 性能 | ✅ 高 (二进制协议) | ⚠️ 中 (JSON 文本) |
| 跨平台 | ✅ 优秀 | ✅ 优秀 |
| 调试 | ⚠️ 较难 | ✅ 容易 |
| 缓存 | ❌ 不支持 | ✅ HTTP 缓存 |
| 版本控制 | ⚠️ 需要手动处理 | ✅ URL 版本化 |

### RPC 最佳实践

1. **服务接口设计**:
   - 保持接口简洁，单一职责
   - 使用清晰的方法命名
   - 避免过多参数，使用数据类封装

2. **错误处理**:
   ```kotlin
   sealed class RpcResult<out T> {
       data class Success<T>(val data: T) : RpcResult<T>()
       data class Error(val message: String, val code: Int) : RpcResult<Nothing>()
   }
   
   interface NewsService {
       suspend fun getArticles(category: String?): RpcResult<List<NewsArticle>>
   }
   ```

3. **日志记录**:
   ```kotlin
   install(KRpc) {
       endpoint("/rpc") {
           service<NewsService> {
               before { call ->
                   println("RPC call: ${call.method}")
               }
               after { call, result ->
                   println("RPC result: ${result.isSuccess}")
               }
           }
       }
   }
   ```

4. **认证授权**:
   ```kotlin
   install(Authentication) {
       jwt("jwt") {
           // JWT 配置
       }
   }
   
   install(KRpc) {
       endpoint("/rpc") {
           authenticate("jwt") {
               service<NewsService> {
                   // 受保护的 RPC 服务
               }
           }
       }
   }
   ```

## 混合集成方案

### RPC + REST 混合使用

在某些场景下，可以同时使用 RPC 和 REST:

```kotlin
// 文件上传使用 REST
routing {
    post("/api/upload") {
        val multipart = call.receiveMultipart()
        // 处理文件上传
    }
}

// 业务逻辑使用 RPC
install(KRpc) {
    endpoint("/rpc") {
        service<FileService> {
            getFileMetadata = { id -> /* ... */ }
            deleteFile = { id -> /* ... */ }
        }
    }
}
```

### WebSocket 实时通信

```kotlin
install(WebSockets) {
    contentConverter = KotlinxWebsocketSerde(Json)
}

routing {
    webSocket("/ws/notifications") {
        val userId = call.parameters["userId"]
        
        // 发送通知
        sendSerialized(Notification(
            userId = userId,
            message = "新消息通知"
        ))
    }
}
```

## 部署配置

### Docker 部署

**Dockerfile**:

```dockerfile
FROM eclipse-temurin:17-jdk-alpine AS build
WORKDIR /app
COPY . .
RUN ./gradlew :server:installDist

FROM eclipse-temurin:17-jdk-alpine
WORKDIR /app
COPY --from=build /app/server/build/install/server /app
EXPOSE 8080
ENTRYPOINT ["java", "-jar", "/app/lib/server.jar"]
```

**docker-compose.yml**:

```yaml
version: '3.8'
services:
  server:
    build: .
    ports:
      - "8080:8080"
    environment:
      - DATABASE_URL=jdbc:postgresql://db:5432/appdb
      - RPC_ENDPOINT=/rpc
      - CORS_ORIGINS=https://yourdomain.com
    depends_on:
      - db
  
  db:
    image: postgres:15
    environment:
      - POSTGRES_DB=appdb
      - POSTGRES_USER=postgres
      - POSTGRES_PASSWORD=secret
    volumes:
      - postgres_data:/var/lib/postgresql/data

volumes:
  postgres_data:
```

## 测试

### RPC 服务测试

```kotlin
import com.example.shared.rpc.NewsService
import com.example.server.rpc.NewsServiceImpl
import kotlinx.coroutines.test.runTest
import kotlin.test.Test
import kotlin.test.assertEquals
import kotlin.test.assertNotNull

class NewsServiceTest {
    private val service = NewsServiceImpl()
    
    @Test
    fun testGetArticles() = runTest {
        val articles = service.getArticles(null)
        assertNotNull(articles)
        assert(articles.isNotEmpty())
    }
    
    @Test
    fun testCreateArticle() = runTest {
        val request = CreateArticleRequest(
            title = "测试文章",
            content = "测试内容",
            author = "测试作者",
            category = "测试"
        )
        
        val article = service.createArticle(request)
        assertEquals("测试文章", article.title)
    }
}
```

### 客户端集成测试

```kotlin
import io.ktor.client.testing.*
import io.ktor.client.plugins.contentnegotiation.*
import kotlinx.serialization.json.Json

class ApiClientTest {
    @Test
    fun testGetUsers() = runTest {
        val client = ApiClient("http://localhost:8080")
        val users = client.getUsers()
        assert(users.isNotEmpty())
    }
}
```

## 参考资源

- [Ktor 官方文档](https://ktor.io/docs/)
- [Kotlin RPC GitHub](https://github.com/Kotlin/kotlinx-rpc)
- [Kotlin Multiplatform 文档](https://kotlinlang.org/docs/multiplatform.html)
- [SQLDelight 文档](https://cashapp.github.io/sqldelight/)
- [Koin 文档](https://insert-koin.io/)
- [Compose Multiplatform 文档](https://www.jetbrains.com/lp/compose-multiplatform/)

## 更新日志

- **2026-03-07**: 重写文档，重点添加 Kotlin Multiplatform 全栈开发和 Kotlin RPC 集成方案
