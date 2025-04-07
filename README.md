# AITransPO

**使用 OpenAI 自动翻译 `.po` 文件内容 (Go 语言实现)**

[![Go Report Card](https://goreportcard.com/badge/github.com/ljtian/AITransPO)](https://goreportcard.com/report/github.com/ljtian/AITransPO) 
[![Status](https://img.shields.io/badge/Status-Development-yellow)](https://github.com/ljtian/AITransPO) 

## 项目简介

`AITransPO(AI Translation for PO)` 是一个使用 Go 语言编写的命令行工具，它利用 OpenAI 的强大语言模型自动翻译 `.po` 文件中的 `msgid` 内容。`.po` 文件是 gettext 标准的本地化文件格式，广泛应用于软件和网站的国际化 (i18n)。本项目旨在帮助开发者和本地化团队快速生成初步翻译，提高本地化效率。

## 主要特性

* **Go 语言实现:** 使用 Go 语言编写，性能高效且易于部署。
* **OpenAI 驱动:** 利用 OpenAI 模型进行高质量的自动翻译。
* **环境变量配置:** 通过环境变量安全地管理 OpenAI API Key。
* **灵活的语言支持:** 支持 OpenAI 模型所支持的多种目标语言。
* **保留 `.po` 文件结构:** 输出文件保持与原始 `.po` 文件相同的结构，包括注释和元数据。
* **处理多行 `msgid`:** 能够正确处理跨越多行的 `msgid` 内容。
* **简单易用:** 提供简单的命令行接口。
* **GitHub Actions 集成:** 方便集成到 CI/CD 流程中，实现自动化翻译。

## 快速上手

### 前提条件

* **Go 语言环境:** 确保你的系统上安装了 Go 语言（建议使用 1.21 或更高版本）。你可以从 [https://go.dev/dl/](https://go.dev/dl/) 下载安装。
* **OpenAI API Key:** 你需要一个有效的 OpenAI API Key。请访问 [https://openai.com/api/](https://openai.com/api/) 注册并获取你的 API Key。

### 安装

1.  **克隆仓库:**

    ```bash
    git clone [https://github.com/ljtian/AITransPO.git](https://github.com/ljtian/AITransPO.git)
    cd AITransPO
    ```

2.  **构建项目:**

    ```bash
    go build -o po_translator .
    ```

    这将在当前目录下生成一个名为 `po_translator` 的可执行文件。

### 配置

1.  **设置 OpenAI API Key:** 将你的 OpenAI API Key 设置为环境变量 `OPENAI_API_KEY`。

    ```bash
    export OPENAI_API_KEY="sk-your-openai-api-key"
    ```
    （请将 `"sk-your-openai-api-key"` 替换为你的实际 API Key）

2.  **配置输入和输出文件:** 默认情况下，程序会读取名为 `my_translations.po` 的输入文件，并将翻译后的内容写入到 `my_translations_zh.po`。你可以在 `main.go` 文件中修改这些常量。

### 使用

1.  **准备 `.po` 文件:** 将你需要翻译的 `.po` 文件放在与可执行文件相同的目录下，或者修改代码中的 `inputFile` 常量指向你的文件。

2.  **运行翻译工具:**

    ```bash
    ./po_translator
    ```

    程序将会读取输入文件，调用 OpenAI API 翻译 `msgid` 中 `msgstr` 为空的条目，并将结果写入到输出文件中。你可以在终端看到翻译的进度和结果。

3.  **检查和校对:** 自动翻译的结果可能需要人工校对和润色，以确保翻译的准确性和自然性。

## GitHub Actions 集成

你可以将 `AI Translation for PO` 集成到你的 GitHub Actions Workflow 中，实现自动化翻译流程。以下是一个示例 Workflow 步骤：

```yaml
name: Translate PO Files

on: [push]

jobs:
  translate:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4
      - name: Set up Go
        uses: actions/setup-go@v5
        with:
          go-version: '1.21'
      - name: Build
        run: go build -o po_translator .
      - name: Translate PO file
        env:
          OPENAI_API_KEY: ${{ secrets.OPENAI_API_KEY }} # 从 GitHub Secrets 中获取
        run: ./po_translator
      - name: Upload translated PO file
        uses: actions/upload-artifact@v3
        with:
          name: translated-po
          path: my_translations_zh.po
```

确保你在 GitHub 仓库的 Secrets 中添加了名为 OPENAI_API_KEY 的 Secret，其值为你的 OpenAI API Key。

## 贡献
欢迎任何形式的贡献！如果你有任何建议、bug 报告或者希望添加新功能，请随时提交 Issue 或 Pull Request。

## 许可证
本项目使用 Apache License 许可证。详细信息请查看 LICENSE 文件。

## 免责声明
本项目使用 OpenAI API 进行翻译，翻译质量取决于 OpenAI 模型的性能。请务必对自动翻译的结果进行人工审核，以确保其准确性和适用性。使用 OpenAI API 可能会产生费用，请注意查阅 OpenAI 的定价策略。
