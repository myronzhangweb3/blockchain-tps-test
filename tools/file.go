package tools

import (
	"bufio"
	"fmt"
	"os"
	"strings"
)

func FileRead(filepath string) (res []string, err error) {
	if !fileExists(filepath) {
		return nil, fmt.Errorf("file not found: %s", filepath)
	}

	file, err := os.Open(filepath)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		// 逐行读取内容
		res = append(res, scanner.Text())
	}

	return
}

func FileWrite(filepath string, pks []string) error {
	if fileExists(filepath) {
		return fmt.Errorf("file already exists: %s", filepath)
	}

	// 创建文件
	file, err := os.Create(filepath)
	if err != nil {
		return err
	}
	defer file.Close()

	// 写入文件
	_, err = file.WriteString(strings.Join(pks, "\n"))
	if err != nil {
		return err
	}

	return nil
}

// fileExists 检查文件是否存在
func fileExists(filepath string) bool {
	info, err := os.Stat(filepath)
	if os.IsNotExist(err) {
		// 文件不存在
		return false
	}
	return !info.IsDir() // 确保是文件而非目录
}
