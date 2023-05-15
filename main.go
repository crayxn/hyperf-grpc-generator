package main

import (
	"flag"
	"fmt"
	"io/ioutil"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

type replacer struct {
	search  string
	replace string
}

var outputPath = flag.String("path", "./", "output file path")
var pluginPath = flag.String("pluginPath", "/usr/local/lib/grpc_php_plugin", "plugin path")
var protoPath = flag.String("protoPath", "", "proto path")
var protoFile = flag.String("proto", "", "proto file")
var clientExtendClass = flag.String("clientExtendClass", "\\Crayoon\\HyperfGrpcClient\\BaseGrpcClient", "php client extend class namespace(option)")

//var phpNamespace = flag.String("phpName", "", "php namespace(option)")
//var phpMetadataNamespace = flag.String("phpMetadataNamespace", "", "php metadata namespace(option)")

func main() {
	flag.Parse()
	//判断是否有安装protoc
	protocPath, err := exec.LookPath("protoc")
	if err != nil {
		fmt.Println("Protoc Not Found!")
		return
	}
	path, _ := filepath.Abs(*outputPath)

	tempPath, _ := os.MkdirTemp(path, "grpc_temp_")
	defer os.RemoveAll(tempPath)

	protoPaths := []string{
		"--php_out=" + tempPath,
		"--grpc_out=" + tempPath,
		"--plugin=protoc-gen-grpc=" + *pluginPath,
	}
	for _, path := range strings.Split(*protoPath, ",") {
		protoPaths = append(protoPaths, "--proto_path="+path)
	}

	for _, path := range strings.Split(*protoFile, ",") {
		protoPaths = append(protoPaths, path)
	}

	cmd := exec.Command(protocPath, protoPaths...)

	out, err := cmd.CombinedOutput()
	if err != nil {
		fmt.Println(cmd.String())
		fmt.Println("Protoc Error:", err)
		return
	}
	fmt.Printf("Protoc Output: %s\n", out)

	//要替换的内容
	replaces := []replacer{
		{
			"/**\n     * @param string $hostname hostname\n     * @param array $opts channel options\n     * @param \\Grpc\\Channel $channel (optional) re-use channel object\n     */\n    public function __construct($hostname, $opts, $channel = null) {\n        parent::__construct($hostname, $opts, $channel);\n    }\n\n    ",
			"",
		},
		{"extends \\Grpc\\BaseStub", "extends " + *clientExtendClass},
		{"@return \\Grpc\\UnaryCall", "@return array"},
	}
	if err := replaceInDir(tempPath, replaces, path); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}

	fmt.Println("All files in the directory and its subdirectories have been replaced successfully!")
}

func replaceInDir(tempDir string, replaces []replacer, targetDir string) error {
	// 遍历指定目录
	err := filepath.Walk(tempDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		if info.IsDir() {
			return nil
		}

		if strings.Contains(path, "Client.php") {
			// 如果是文件则进行替换操作
			if err := replaceInFile(path, replaces); err != nil {
				return err
			}
		}

		relPath := strings.TrimPrefix(path, tempDir)
		targetPath := strings.Replace(filepath.Join(targetDir, relPath), "/App/", "/app/", 1)
		//创建需要的路径
		if err := os.MkdirAll(filepath.Dir(targetPath), 0777); err != nil {
			return err
		}
		//移入指定路径
		if err := os.Rename(path, targetPath); err != nil {
			return err
		}

		return nil
	})
	if err != nil {
		return err
	}

	return nil
}

func replaceInFile(fileName string, replaces []replacer) error {
	// 读取整个文件作为字符串
	bytes, err := ioutil.ReadFile(fileName)
	if err != nil {
		return err
	}
	fileContent := string(bytes)

	// 检查文件内容中是否包含构造函数
	if !strings.Contains(fileContent, "__construct") {
		return nil // 如果文件中不包含所需的字符串，则忽略该文件并继续遍历
	}

	// 创建新文件来写入替换后的内容
	tempFile, err := os.Create(fileName + ".temp")
	if err != nil {
		return err
	}

	defer func() {
		tempFile.Close()
		os.Remove(fileName)
		os.Rename(fileName+".temp", fileName)
	}()

	for _, replacer := range replaces {
		fileContent = strings.Replace(fileContent, replacer.search, replacer.replace, -1)
	}
	if _, err := tempFile.WriteString(fileContent); err != nil {
		return err
	}

	return nil
}
