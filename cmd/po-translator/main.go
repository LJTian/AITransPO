package main

import (
	"bufio"
	"fmt"
	"log"
	"os"
	"regexp"
	"strings"

	"github.com/spf13/cobra"

	"github.com/ljtian/aitranspo/pkg/translator"
)

var (
	inputFile      string
	outputFile     string
	targetLanguage string
	openaiModel    string
	openaiAPIKey   string
)

var (
	msgidRegex  = regexp.MustCompile(`msgid "(.*?)"`)
	msgstrRegex = regexp.MustCompile(`msgstr "(.*?)"`)
)

var rootCmd = &cobra.Command{
	Use:   "po-translator",
	Short: "使用 OpenAI 自动翻译 .po 文件",
	Long:  `po-translator 是一个命令行工具，用于使用 OpenAI 模型自动翻译 .po 文件。`,
}

func translateAndWrite(msgid string, outputWriter *bufio.Writer, targetLanguage, openaiAPIKey, openaiModel string,
	commonErrors map[string]bool) (translated bool, err error) {

	//log.Printf("翻译内容: [%v]\n", msgid)
	strings.TrimSpace(msgid)
	if msgid == "" {
		outputWriter.WriteString("msgstr \"\"\n")
		return false, nil
	}
	translatedText, transErr := translator.TranslateWithOpenAI(msgid, targetLanguage, openaiAPIKey, openaiModel)
	if transErr != nil {
		log.Printf("翻译 '%s' 失败: %v\n", msgid, transErr)
		return false, nil
	}
	if _, found := commonErrors[strings.TrimSpace(translatedText)]; found {
		log.Printf("翻译 '%s' 结果包含常见错误，跳过: '%s'\n", msgid, translatedText)
		return false, nil
	}
	if len(translatedText) > len(msgid)*4 {
		log.Printf("翻译 '%s' 结果超过最大长度 (%d)，跳过\n", msgid, len(msgid)*4)
		outputWriter.WriteString("msgstr \"\"\n")
		return false, nil
	}
	_, err = outputWriter.WriteString(fmt.Sprintf("msgstr \"%s\"\n", translatedText))
	if err != nil {
		return true, fmt.Errorf("写入输出文件失败: %v", err)
	}
	return true, nil
}

func processSingleLineMsgID(line string, inputScanner *bufio.Scanner, outputWriter *bufio.Writer,
	targetLanguage, openaiAPIKey, openaiModel string, commonErrors map[string]bool) (translated bool, skipped bool, err error) {
	match := msgidRegex.FindStringSubmatch(line)
	if len(match) > 1 {
		currentMsgid := match[1]
		_, writeErr := outputWriter.WriteString(line + "\n") // 先写入 msgid
		if writeErr != nil {
			return false, false, fmt.Errorf("写入输出文件失败: %v", writeErr)
		}
		if inputScanner.Scan() {
			nextLine := inputScanner.Text()
			if strings.HasPrefix(nextLine, "msgstr \"\"") {
				translated, err = translateAndWrite(currentMsgid, outputWriter, targetLanguage, openaiAPIKey,
					openaiModel, commonErrors)
				if err != nil {
					//outputWriter.WriteString("msgstr \"\"\n")
					return translated, false, err
				}

				// 需要额外读取一个空行（.po 文件中 msgstr 后通常会有一个空行）
				if inputScanner.Scan() {
					_, writeErr := outputWriter.WriteString(inputScanner.Text() + "\n")
					if writeErr != nil {
						return translated, false, fmt.Errorf("写入输出文件失败: %v", writeErr)
					}
				}
			} else if strings.HasPrefix(nextLine, "msgstr \"") {
				skipped = true
				log.Printf("跳过单行 msgid '%s'，已有 msgstr: '%s'", currentMsgid, nextLine) // 添加日志
				_, writeErr := outputWriter.WriteString(nextLine + "\n")              // 写入已有的 msgstr
				if writeErr != nil {
					return false, true, fmt.Errorf("写入输出文件失败: %v", writeErr)
				}
			} else {
				_, writeErr := outputWriter.WriteString(nextLine + "\n") // 写入其他行
				if writeErr != nil {
					return false, false, fmt.Errorf("写入输出文件失败: %v", writeErr)
				}
			}
		}
	}
	return
}

func processMultilineMsgID(firstLine string, inputScanner *bufio.Scanner, outputWriter *bufio.Writer, targetLanguage, openaiAPIKey, openaiModel string, commonErrors map[string]bool) (translated bool, skipped bool, err error) {
	var fullMsgid strings.Builder
	var msgidLines []string
	msgidLines = append(msgidLines, firstLine) // 先保存第一行 msgid ""

	for inputScanner.Scan() {
		nextLine := inputScanner.Text()
		//log.Printf("nextLine is [%s]\n", nextLine)
		if strings.HasPrefix(nextLine, "\"") && strings.HasSuffix(nextLine, "\"") {
			fullMsgid.WriteString(nextLine[1 : len(nextLine)-1])
			msgidLines = append(msgidLines, nextLine)
		} else if strings.HasPrefix(nextLine, "msgstr \"\"") {
			// 写入 msgid 的所有行
			for _, idLine := range msgidLines {
				_, writeErr := outputWriter.WriteString(idLine + "\n")
				if writeErr != nil {
					return false, false, fmt.Errorf("写入输出文件失败: %v", writeErr)
				}
			}
			translated, err = translateAndWrite(fullMsgid.String(), outputWriter, targetLanguage, openaiAPIKey, openaiModel, commonErrors)
			if err != nil {
				//outputWriter.WriteString("msgstr \"\"\n")
				return translated, false, err
			}
			// 需要额外读取一个空行
			if inputScanner.Scan() {
				_, writeErr := outputWriter.WriteString(inputScanner.Text() + "\n")
				if writeErr != nil {
					return translated, false, fmt.Errorf("写入输出文件失败: %v", writeErr)
				}
			}
			break
		} else if strings.HasPrefix(nextLine, "msgstr \"") {
			skipped = true
			log.Printf("跳过多行 msgid (first line: '%s')，已有 msgstr: '%s'", firstLine, nextLine) // 添加日志
			// 写入 msgid 的所有行
			for _, idLine := range msgidLines {
				_, writeErr := outputWriter.WriteString(idLine + "\n")
				if writeErr != nil {
					return false, true, fmt.Errorf("写入输出文件失败: %v", writeErr)
				}
			}
			_, writeErr := outputWriter.WriteString(nextLine + "\n") // 写入已有的 msgstr
			if writeErr != nil {
				return false, true, fmt.Errorf("写入输出文件失败: %v", writeErr)
			}
			break
		}
	}
	return
}

func processPOFile(inputFile, outputFile, targetLanguage, openaiAPIKey, openaiModel string, commonErrors map[string]bool) (translatedCount int, errorCount int, skippedCount int, err error) {
	inputFileHandle, err := os.Open(inputFile)
	if err != nil {
		return 0, 0, 0, fmt.Errorf("无法打开输入文件 '%s': %v", inputFile, err)
	}
	defer inputFileHandle.Close()

	outputFileHandle, err := os.Create(outputFile)
	if err != nil {
		return 0, 0, 0, fmt.Errorf("无法创建输出文件 '%s': %v", outputFile, err)
	}
	defer outputFileHandle.Close()

	inputScanner := bufio.NewScanner(inputFileHandle)
	outputWriter := bufio.NewWriter(outputFileHandle)
	defer outputWriter.Flush()

	fmt.Println("开始处理翻译...")
	translatedCount = 0
	skippedCount = 0

	for inputScanner.Scan() {
		line := inputScanner.Text()

		if strings.HasPrefix(line, "msgid \"\"") {
			translated, skipped, procErr := processMultilineMsgID(line, inputScanner, outputWriter, targetLanguage, openaiAPIKey, openaiModel, commonErrors)
			if procErr != nil {
				return translatedCount, errorCount, skippedCount, procErr
			}
			if translated {
				translatedCount++
			}
			if skipped {
				skippedCount++
			}
		} else if strings.HasPrefix(line, "msgid \"") {
			translated, skipped, procErr := processSingleLineMsgID(line, inputScanner, outputWriter, targetLanguage, openaiAPIKey, openaiModel, commonErrors)
			if procErr != nil {
				return translatedCount, errorCount, skippedCount, procErr
			}
			if translated {
				translatedCount++
			}
			if skipped {
				skippedCount++
			}
		} else {
			_, writeErr := outputWriter.WriteString(line + "\n")
			if writeErr != nil {
				return translatedCount, errorCount, skippedCount, fmt.Errorf("写入输出文件失败: %v", writeErr)
			}
		}
	}

	if err := inputScanner.Err(); err != nil {
		return translatedCount, errorCount, skippedCount, fmt.Errorf("读取输入文件出错: %v", err)
	}

	return translatedCount, errorCount, skippedCount, nil
}

// GetRootCmd returns the root command
func GetRootCmd() *cobra.Command {
	return rootCmd
}

func main() {

	commonErrors := map[string]bool{
		"":     true,
		" ":    true,
		"翻译失败": true,
		"您接受的培训数据截至2023年10月。": true,
		// 添加更多你认为常见的错误翻译
	}

	rootCmd.Run = func(cmd *cobra.Command, args []string) {
		translatedCount, errorCount, skippedCount, err := processPOFile(inputFile, outputFile, targetLanguage, openaiAPIKey, openaiModel, commonErrors)
		if err != nil {
			log.Fatalf("处理 .po 文件失败: %v", err)
		}
		fmt.Printf("\n处理完成。\n翻译条目数: %d\n因长度限制跳过条目数: %d\n跳过已翻译条目数: %d\n", translatedCount, errorCount, skippedCount)
	}

	if err := rootCmd.Execute(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}

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
