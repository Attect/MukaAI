# Task 3: 依赖注入容器

## 元数据
- **标题**: 依赖注入容器实现
- **作者**: 开发团队
- **日期**: 2026-04-06
- **版本**: 1.0.0

## 题目描述

实现一个轻量级的依赖注入（Dependency Injection, DI）容器，类似于Spring IoC容器的核心功能。该容器需要支持构造器注入和字段注入、单例和原型作用域、循环依赖检测等功能，帮助开发者理解依赖注入的核心原理。

## 功能需求

### 1. 核心功能

#### 1.1 Bean定义与注册
- 支持通过API注册Bean定义
- 支持通过注解扫描自动注册Bean
- 支持Bean的名称和类型注册
- 支持Bean的别名

#### 1.2 依赖注入方式
- **构造器注入**: 通过构造方法注入依赖
- **字段注入**: 通过反射直接注入字段
- **Setter方法注入**: 通过Setter方法注入依赖
- 支持混合注入方式

#### 1.3 作用域管理
- **单例（Singleton）**: 整个容器中只有一个实例
- **原型（Prototype）**: 每次获取都创建新实例
- 支持自定义作用域（如Request、Session等）

#### 1.4 生命周期管理
- 支持Bean的初始化回调（@PostConstruct）
- 支持Bean的销毁回调（@PreDestroy）
- 支持BeanNameAware、ApplicationContextAware等感知接口

#### 1.5 循环依赖检测
- 检测构造器注入的循环依赖（直接抛出异常）
- 处理字段注入的循环依赖（通过代理或延迟注入）
- 提供清晰的错误信息

#### 1.6 条件化注册
- 支持基于条件的Bean注册（@Conditional）
- 支持基于配置的Bean注册
- 支持Profile环境区分

### 2. 技术要求

#### 2.1 设计模式应用
- **工厂模式**: Bean的创建和管理
- **单例模式**: 单例作用域的实现
- **代理模式**: 循环依赖的处理
- **模板方法模式**: Bean生命周期的管理

#### 2.2 反射与注解
- 使用Java反射API进行依赖注入
- 支持自定义注解（@Component、@Autowired等）
- 支持注解的继承和组合

#### 2.3 性能优化
- 支持延迟初始化（@Lazy）
- 缓存反射结果
- 优化Bean创建性能

## API设计示例

```java
// 创建容器
ApplicationContext context = new AnnotationConfigApplicationContext();

// 方式1: 通过API注册Bean
context.registerBean(UserService.class, BeanScope.SINGLETON);
context.registerBean(OrderService.class, BeanScope.PROTOTYPE);
context.registerBean("dataSource", DataSource.class, () -> new DataSource());

// 方式2: 通过注解扫描
context.scan("com.example.service");

// 刷新容器
context.refresh();

// 获取Bean
UserService userService = context.getBean(UserService.class);
OrderService orderService = context.getBean("orderService", OrderService.class);

// 使用注解定义Bean
@Component
@Scope(BeanScope.SINGLETON)
public class UserService {
    @Autowired
    private OrderService orderService;
    
    @PostConstruct
    public void init() {
        // 初始化逻辑
    }
    
    @PreDestroy
    public void destroy() {
        // 销毁逻辑
    }
}

// 构造器注入
@Component
public class OrderService {
    private final UserService userService;
    
    @Autowired
    public OrderService(UserService userService) {
        this.userService = userService;
    }
}

// 条件化注册
@Component
@Conditional(OnPropertyCondition.class)
public class FeatureService {
    // 仅当配置满足条件时才注册
}

// 自定义条件
public class OnPropertyCondition implements Condition {
    @Override
    public boolean matches(ConditionContext context) {
        return context.getEnvironment().getProperty("feature.enabled", "false").equals("true");
    }
}
```

## 验收标准

### 功能验收

| 序号 | 验收项 | 验收标准 | 优先级 |
|------|--------|----------|--------|
| 1 | Bean注册 | 能够通过API成功注册Bean定义 | P0 |
| 2 | 构造器注入 | 能够正确注入构造器参数 | P0 |
| 3 | 字段注入 | 能够正确注入字段依赖 | P0 |
| 4 | 单例作用域 | 单例Bean在容器中只有一个实例 | P0 |
| 5 | 原型作用域 | 原型Bean每次获取都创建新实例 | P0 |
| 6 | 循环依赖检测 | 能够检测并报告构造器循环依赖 | P0 |
| 7 | 循环依赖处理 | 能够处理字段注入的循环依赖 | P0 |
| 8 | 生命周期回调 | 初始化和销毁回调正确执行 | P1 |
| 9 | 延迟初始化 | 延迟初始化的Bean按需创建 | P1 |
| 10 | 条件化注册 | 条件满足时才注册Bean | P1 |

### 质量验收

| 序号 | 验收项 | 验收标准 |
|------|--------|----------|
| 1 | 单元测试 | 测试覆盖率 ≥ 80% |
| 2 | 性能要求 | 单例Bean获取延迟 < 1ms |
| 3 | 性能要求 | 原型Bean创建延迟 < 10ms |
| 4 | 代码规范 | 通过阿里巴巴Java开发规范检查 |
| 5 | 文档完整 | 包含设计文档、API文档、使用示例 |

## 测试用例

### 测试场景1: 构造器注入
```java
@Component
public class ServiceA {
    private final ServiceB serviceB;
    
    @Autowired
    public ServiceA(ServiceB serviceB) {
        this.serviceB = serviceB;
    }
}

// 验证ServiceA的serviceB字段被正确注入
```

### 测试场景2: 字段注入
```java
@Component
public class ServiceA {
    @Autowired
    private ServiceB serviceB;
}

// 验证ServiceA的serviceB字段被正确注入
```

### 测试场景3: 单例作用域
```java
@Component
@Scope(BeanScope.SINGLETON)
public class SingletonService {}

// 多次获取，验证返回同一实例
SingletonService instance1 = context.getBean(SingletonService.class);
SingletonService instance2 = context.getBean(SingletonService.class);
assert instance1 == instance2;
```

### 测试场景4: 原型作用域
```java
@Component
@Scope(BeanScope.PROTOTYPE)
public class PrototypeService {}

// 多次获取，验证返回不同实例
PrototypeService instance1 = context.getBean(PrototypeService.class);
PrototypeService instance2 = context.getBean(PrototypeService.class);
assert instance1 != instance2;
```

### 测试场景5: 循环依赖检测
```java
@Component
public class ServiceA {
    @Autowired
    public ServiceA(ServiceB serviceB) {}
}

@Component
public class ServiceB {
    @Autowired
    public ServiceB(ServiceA serviceA) {}
}

// 验证抛出循环依赖异常
```

### 测试场景6: 字段注入循环依赖
```java
@Component
public class ServiceA {
    @Autowired
    private ServiceB serviceB;
}

@Component
public class ServiceB {
    @Autowired
    private ServiceA serviceA;
}

// 验证能够正确处理循环依赖
```

### 测试场景7: 生命周期回调
```java
@Component
public class LifecycleService {
    private boolean initialized = false;
    private boolean destroyed = false;
    
    @PostConstruct
    public void init() {
        initialized = true;
    }
    
    @PreDestroy
    public void destroy() {
        destroyed = true;
    }
}

// 验证初始化和销毁回调被正确调用
```

### 测试场景8: 延迟初始化
```java
@Component
@Lazy
public class LazyService {}

// 验证容器启动时不创建实例
// 验证第一次获取时才创建实例
```

## 项目结构

```
task3-dependency-injection/
├── src/
│   ├── main/
│   │   └── java/
│   │       └── com/di/
│   │           ├── core/              # 核心类
│   │           │   ├── ApplicationContext.java
│   │           │   ├── BeanDefinition.java
│   │           │   ├── BeanFactory.java
│   │           │   └── BeanRegistry.java
│   │           ├── context/           # 上下文
│   │           │   ├── AnnotationConfigApplicationContext.java
│   │           │   └── Environment.java
│   │           ├── injector/          # 注入器
│   │           │   ├── Injector.java
│   │           │   ├── ConstructorInjector.java
│   │           │   └── FieldInjector.java
│   │           ├── scope/             # 作用域
│   │           │   ├── Scope.java
│   │           │   ├── SingletonScope.java
│   │           │   └── PrototypeScope.java
│   │           ├── lifecycle/         # 生命周期
│   │           │   ├── LifecycleProcessor.java
│   │           │   └── LifecycleAware.java
│   │           ├── scanner/           # 扫描器
│   │           │   └── ClassPathScanner.java
│   │           ├── condition/         # 条件
│   │           │   ├── Condition.java
│   │           │   └── ConditionEvaluator.java
│   │           └── annotation/        # 注解
│   │               ├── Component.java
│   │               ├── Autowired.java
│   │               ├── Scope.java
│   │               ├── Lazy.java
│   │               ├── PostConstruct.java
│   │               ├── PreDestroy.java
│   │               └── Conditional.java
│   └── test/
│       └── java/
│           └── com/di/
│               ├── ApplicationContextTest.java
│               ├── InjectionTest.java
│               ├── ScopeTest.java
│               ├── CircularDependencyTest.java
│               ├── LifecycleTest.java
│               └── PerformanceTest.java
├── README.md
└── pom.xml
```

## 提交清单

- [ ] 完整的源代码实现
- [ ] 单元测试代码（覆盖率 ≥ 80%）
- [ ] 设计文档（包含类图、时序图）
- [ ] API使用示例
- [ ] 性能测试报告
- [ ] 代码规范检查报告

## 参考资料

- [Spring Framework IoC容器源码](https://github.com/spring-projects/spring-framework/tree/main/spring-beans)
- [依赖注入模式详解](https://martinfowler.com/articles/injection.html)
- [Java反射教程](https://docs.oracle.com/javase/tutorial/reflect/)
