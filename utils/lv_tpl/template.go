/*
 * Copyright 2019 lostvip
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *     http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

package lv_tpl

import (
	"bytes"
	"fmt"
	"github.com/lostvip-com/lv_framework/utils/lv_secret"
	"os"
	"path/filepath"
	"strings"
	"text/template"
)

// 读取模板
func ParseTemplate(templateName string, data interface{}) (string, error) {
	cur, err := os.Getwd()
	if err != nil {
		return "", err
	}
	templatePath := filepath.Join(cur, "template", templateName)
	b, err := os.ReadFile(templatePath)
	if err != nil {
		return "", err
	}
	templateStr := string(b)

	tmpl, err := template.New(templateName).Parse(templateStr) //建立一个模板，内容是"hello, {{OssUrl}}"
	if err != nil {
		return "", fmt.Errorf("failed to parse template %s: %w", templateName, err)
	}
	buffer := bytes.NewBufferString("")
	err = tmpl.Execute(buffer, data) //将string与模板合成，变量name的内容会替换掉{{OssUrl}}
	if err != nil {
		return "", err
	}
	return buffer.String(), err
}

// 读取模板
func ParseTemplateStr(templateStr string, data interface{}) (string, error) {
	templateName := lv_secret.Md5(templateStr)
	tmpl, err := template.New(templateName).Parse(templateStr) //建立一个模板，内容是"hello, {{OssUrl}}"
	if err != nil {
		return "", err
	}
	buffer := bytes.NewBufferString("")
	err = tmpl.Execute(buffer, data) //替换模板变量
	if err != nil {
		return "", err
	}
	str := strings.ReplaceAll(buffer.String(), "\n", " ")
	return str, err
}
