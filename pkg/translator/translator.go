package translator

import (
	"bufio"
	"bytes"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"regexp"
	"strings"
)

const openaiAPIEndpoint = "https://api.openai.com/v1/chat/completions"

// LoadPOFile 从 .po 文件加载翻译条目
func LoadPOFile(filePath string) (map[string]string, error) {
	translations := make(map[string]string)
	file, err := os.Open(filePath)
	if err != nil {
		return nil, fmt.Errorf("无法打开 .po 文件 '%s': %v", filePath, err)
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	var currentMsgid string
	var multilineMsgid strings.Builder
	inMultilineMsgid := false

	for scanner.Scan() {
		line := scanner.Text()
		if strings.HasPrefix(line, "msgid \"") {
			re := regexp.MustCompile(`msgid "(.*?)"`)
			match := re.FindStringSubmatch(line)
			if len(match) > 1 {
				currentMsgid = match[1]
				inMultilineMsgid = false
				multilineMsgid.Reset()
			} else {
				inMultilineMsgid = true
				multilineMsgid.Reset()
			}
		} else if strings.HasPrefix(line, "msgstr \"") {
			re := regexp.MustCompile(`msgstr "(.*?)"`)
			match := re.FindStringSubmatch(line)
			if len(match) > 1 {
				translations[currentMsgid] = match[1]
			}
			currentMsgid = ""
		} else if strings.HasPrefix(line, "\"") && strings.HasSuffix(line, "\"") && inMultilineMsgid {
			multilineMsgid.WriteString(line[1 : len(line)-1])
			currentMsgid = multilineMsgid.String()
		} else if strings.HasPrefix(line, "msgstr \"\"") {
			translations[currentMsgid] = "" // 标记为空，需要翻译
		}
	}
	if err := scanner.Err(); err != nil {
		log.Println("读取 .po 文件出错:", err)
	}
	return translations, nil
}

// WritePOFile 将翻译后的内容写回 .po 文件
func WritePOFile(originalFile string, translations map[string]string, outputFile string) error {
	originalContent, err := os.ReadFile(originalFile)
	if err != nil {
		return fmt.Errorf("无法读取原始 .po 文件 '%s': %v", originalFile, err)
	}
	lines := strings.Split(string(originalContent), "\n")
	var outputLines []string

	var currentMsgid string
	inMultilineMsgid := false
	var multilineMsgidContent strings.Builder

	for _, line := range lines {
		outputLines = append(outputLines, line)

		if strings.HasPrefix(line, "msgid \"") {
			re := regexp.MustCompile(`msgid "(.*?)"`)
			match := re.FindStringSubmatch(line)
			if len(match) > 1 {
				currentMsgid = match[1]
				outputLines = append(outputLines, fmt.Sprintf("msgstr \"%s\"", translations[currentMsgid]))
			} else {
				inMultilineMsgid = true
				multilineMsgidContent.Reset()
			}
		} else if strings.HasPrefix(line, "msgid \"\"") {
			inMultilineMsgid = true
			multilineMsgidContent.Reset()
		} else if inMultilineMsgid && strings.HasPrefix(line, "\"") && strings.HasSuffix(line, "\"") {
			multilineMsgidContent.WriteString(line[1 : len(line)-1])
			currentMsgid = multilineMsgidContent.String()
		} else if inMultilineMsgid && strings.HasPrefix(line, "msgstr \"\"") {
			outputLines = append(outputLines, fmt.Sprintf("msgstr \"%s\"", translations[currentMsgid]))
			inMultilineMsgid = false
			multilineMsgidContent.Reset()
		} else if strings.HasPrefix(line, "msgstr") {
			continue // Skip original msgstr
		}
	}

	outputContent := strings.Join(outputLines, "\n")
	err = os.WriteFile(outputFile, []byte(outputContent), 0644)
	if err != nil {
		return fmt.Errorf("无法写入翻译后的 .po 文件 '%s': %v", outputFile, err)
	}
	return nil
}

// TranslateWithOpenAI 使用 OpenAI API 进行翻译
func TranslateWithOpenAI(text, targetLanguage, apiKey, model string) (string, error) {
	messages := []map[string]string{
		{"role": "system", "content": fmt.Sprintf("将以下文本翻译成 %s.", targetLanguage)},
		{"role": "user", "content": text},
	}

	requestBody, err := json.Marshal(map[string]interface{}{
		"model":    model,
		"messages": messages,
	})
	if err != nil {
		return "", err
	}

	req, err := http.NewRequest("POST", openaiAPIEndpoint, bytes.NewBuffer(requestBody))
	if err != nil {
		return "", err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", apiKey))

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("OpenAI API 请求失败，状态码: %d", resp.StatusCode)
	}

	var response struct {
		Choices []struct {
			Message struct {
				Content string `json:"content"`
			} `json:"message"`
		} `json:"choices"`
	}

	err = json.NewDecoder(resp.Body).Decode(&response)
	if err != nil {
		return "", err
	}

	if len(response.Choices) > 0 {
		return response.Choices[0].Message.Content, nil
	}

	return "", fmt.Errorf("OpenAI API 响应中没有翻译结果")
}