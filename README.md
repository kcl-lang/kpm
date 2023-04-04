# kpm
KCL Package Manager

```go
package main

import (
	"bytes"
	"io/ioutil"
	"mime/multipart"
	"net/http"
)

func main() {
	// 读取 tar 文件内容
	data, err := ioutil.ReadFile("image.tar")
	if err != nil {
		panic(err)
	}

	// 创建 multipart/form-data 编码的请求体
	var b bytes.Buffer
	w := multipart.NewWriter(&b)
	fw, err := w.CreateFormFile("file", "image.tar")
	if err != nil {
		panic(err)
	}
	if _, err := fw.Write(data); err != nil {
		panic(err)
	}
	w.Close()

	// 创建 HTTP 请求
	req, err := http.NewRequest("POST", "https://hub.docker.com/v2/repositories/your-namespace/your-repo/images/upload/", &b)
	if err != nil {
		panic(err)
	}
	req.Header.Set("Content-Type", w.FormDataContentType())
	req.Header.Set("Authorization", "Bearer your-token")

	// 发送 HTTP 请求
	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		panic(err)
	}
	defer resp.Body.Close()

	// 处理响应
	// ...
}

```