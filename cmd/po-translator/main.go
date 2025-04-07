package main

import (
	"fmt"
	"log"
	"os"

	"github.com/ljtian/aitranspo/pkg/translator"
	"github.com/spf13/cobra"
)

var (
	inputFile      string
	outputFile     string
	targetLanguage string
	openaiModel    string
	openaiAPIKey   string

	rootCmd = &cobra.Command{
		Use:   "po-translator",
		Short: "使用 OpenAI 自动翻译 .po 文件",
		Long:  `po-translator 是一个命令行工具，用于使用 OpenAI 模型自动翻译 .po 文件。`,
		Run: func(cmd *cobra.Command, args []string) {
			if openaiAPIKey == "" {
				log.Fatal("请设置 OPENAI_API_KEY 环境变量或使用 --openai-api-key 标志")
				return
			}

			translationMap, err := translator.LoadPOFile(inputFile)
			if err != nil {
				log.Fatalf("加载 .po 文件失败: %v", err)
			}

			fmt.Println("需要翻译的条目：")
			for msgid, msgstr := range translationMap {
				if msgstr == "" {
					fmt.Printf("原文 (msgid): %s\n", msgid)
					translatedText, err := translator.TranslateWithOpenAI(msgid, targetLanguage, openaiAPIKey, openaiModel)
					if err != nil {
						log.Printf("翻译 '%s' 失败: %v\n", msgid, err)
						translationMap[msgid] = "" // 翻译失败则保留为空
					} else {
						translationMap[msgid] = translatedText
						fmt.Printf("译文 (msgstr): %s\n", translatedText)
					}
				}
			}

			err = translator.WritePOFile(inputFile, translationMap, outputFile)
			if err != nil {
				log.Fatalf("写入翻译后的 .po 文件失败: %v", err)
			}
			fmt.Printf("\n翻译后的内容已写入到文件: %s\n", outputFile)
		},
	}
)

func init() {
	rootCmd.PersistentFlags().StringVarP(&inputFile, "input", "i", "my_translations.po", "要翻译的 .po 文件路径")
	rootCmd.PersistentFlags().StringVarP(&outputFile, "output", "o", "my_translations_zh.po", "翻译后的 .po 文件输出路径")
	rootCmd.PersistentFlags().StringVarP(&targetLanguage, "target-lang", "t", "zh-CN", "目标语言代码 (例如: zh-CN, en)")
	rootCmd.PersistentFlags().StringVar(&openaiModel, "model", "gpt-3.5-turbo", "要使用的 OpenAI 模型名称")
	rootCmd.PersistentFlags().StringVar(&openaiAPIKey, "openai-api-key", "", "你的 OpenAI API Key (或者设置 OPENAI_API_KEY 环境变量)")

	// 绑定环境变量到标志 (如果标志没有被设置，则使用环境变量的值)
	rootCmd.PersistentFlags().StringVar(&openaiAPIKey, "openai-api-key-env", os.Getenv("OPENAI_API_KEY"), "你的 OpenAI API Key (从环境变量获取)")
	rootCmd.PersistentFlags().MarkHidden("openai-api-key-env") // 隐藏这个辅助标志

	// 优先使用命令行标志的值，如果为空则使用环境变量
	rootCmd.PreRunE = func(cmd *cobra.Command, args []string) error {
		if openaiAPIKey == "" {
			openaiAPIKey = os.Getenv("OPENAI_API_KEY")
			if openaiAPIKey == "" {
				return fmt.Errorf("请提供 OpenAI API Key (通过 --openai-api-key 标志或 OPENAI_API_KEY 环境变量)")
			}
		}
		return nil
	}
}

// GetRootCmd returns the root command
func GetRootCmd() *cobra.Command {
	return rootCmd
}

func main() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}
