# 浏览器控制能力测试方案

## 测试说明
由于 Kotlin 没有官方的 Playwright 绑定，本测试验证通过 Node.js 调用 Playwright 的可行性。

## 前置要求
1. 安装 Node.js (v18+)
2. 安装 Playwright: `npm install -g playwright`
3. 安装浏览器：`npx playwright install chromium`

## 测试脚本
创建一个 Node.js 脚本作为浏览器控制的后台服务。

## Kotlin 集成方式
Kotlin 通过 ProcessBuilder 调用 Node.js 脚本，并捕获输出。

## 备选方案
如果 Playwright 不可用，可以使用:
1. Selenium WebDriver (Java 生态成熟)
2. Chrome DevTools Protocol (CDP) 直接调用
