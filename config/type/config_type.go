package config_type

import (
	"fmt"
	"github.com/Orange0224/go-injector-yaml/config/utils"
	"io/ioutil"
	"log"
	"os"
	"runtime"
	"strings"
)

type TypeScanner struct {
	fileSeparator string
	ConfigDir     string
	importMap     map[string]int
	typeMap       map[string][]string
	orderMap      map[string]int
	typeAlias     map[string]string
}

func (t *TypeScanner) Begin() {
	t.checkConfig()
	t.initVariable()
	t.scanTypeInfo(t.ConfigDir)
	topType := t.getTopType()
	applicationDefinition := t.combinationType(topType)
	fmt.Println(applicationDefinition)
	t.writeResultToFile(applicationDefinition, t.ConfigDir+t.fileSeparator+"config")
}

func (t *TypeScanner) checkConfig() {
	if utils.IsBlank(t.ConfigDir) {
		panic("cannot scan empty path")
	}
}
func (t *TypeScanner) writeResultToFile(result, configPath string) {
	file, err := os.Create(configPath + t.fileSeparator + "config.go")
	if err != nil {
		fmt.Println("Cannot create file")
		return
	}
	defer file.Close()
	file.Write([]byte(result))
	file.Write([]byte("\n"))
}

func (t *TypeScanner) combinationType(topType []string) string {
	header := `
package config
type ApplicationConfig struct{
`
	types := ""
	for i := range topType {
		types += strings.ToUpper(t.typeAlias[topType[i]][0:1]) + t.typeAlias[topType[i]][1:] + "  " + topType[i] + " `yaml:\"" + t.typeAlias[topType[i]] + "\"`" + "\n"
	}
	footer := `}
var applicationConfig ApplicationConfig

func GetApplicationConfig() ApplicationConfig{
return applicationConfig
}
`
	return header + types + footer
}

func (t *TypeScanner) initVariable() {
	if runtime.GOOS == "windows" {
		t.fileSeparator = "\\"
	} else {
		t.fileSeparator = "/"
	}
	t.importMap = make(map[string]int, 0)
	t.typeMap = make(map[string][]string)
	t.orderMap = make(map[string]int)
	t.typeAlias = make(map[string]string)
}

func (t *TypeScanner) scanTypeInfo(dir string) {
	files, _ := t.GetScanFiles(dir + t.fileSeparator + "config")
	for i := range files {
		filename := files[i][strings.LastIndex(files[i], t.fileSeparator)+1:]
		if filename == "config_loader.go" {
			continue
		}
		types, imports, _, aliasMap := t.GetConfigurations(files[i])
		for _, imp := range imports {
			t.importMap[imp] = 1
		}
		for _, typ := range types {
			name := t.GetTypeName(typ[0])
			t.typeMap[name] = typ
			t.orderMap[name] = 0
		}
		for typ, value := range aliasMap {
			t.typeAlias[typ] = value
		}
	}
}

func (t *TypeScanner) getTopType() []string {
	for typ1 := range t.typeMap {
		for typ2 := range t.typeMap {
			if typ1 == typ2 {
				continue
			}
			if t.ContainsType(typ1, typ2) {
				t.orderMap[typ2] = -1
			}
		}
	}
	top := make([]string, 0)
	for k, v := range t.orderMap {
		if v == 0 {
			top = append(top, k)
		}
	}
	return top
}

func (t *TypeScanner) ContainsType(type1, type2 string) bool {
	typeInfo := t.typeMap[type1]
	for i := range typeInfo {
		strs := strings.Split(typeInfo[i], " ")
		for j := range strs {
			if strs[j] == type2 {
				return true
			}
		}
	}
	return false
}

func (t *TypeScanner) GetTypeName(line string) string {
	typeIndex := strings.Index(line, "type ")
	structIndex := strings.Index(line, "struct ")
	if typeIndex != -1 && structIndex != -1 && typeIndex < structIndex {
		return strings.TrimSpace(line[typeIndex+len("type ") : structIndex])
	}
	return ""
}

func (t *TypeScanner) GetTypeAlias(line, typeName string) string {
	index := strings.Index(line, "@Alias=")
	if index == -1 {
		return strings.ToLower(typeName[0:1]) + typeName[1:]
	}
	split := strings.Split(line[index+len("@Alias="):], " ")
	return split[0]
}

func (t *TypeScanner) GetScanFiles(root string) (files, dirs []string) {
	//获取文件或目录相关信息
	fileInfoList, err := ioutil.ReadDir(root)
	if err != nil {
		log.Fatal(err)
		return
	}
	for _, item := range fileInfoList {
		if item.IsDir() && item.Name()[0] != '.' {
			dirs = append(dirs, item.Name())
		} else {
			if strings.Index(item.Name(), ".go") != -1 {
				files = append(files, item.Name())
			}
		}
	}
	for i := range files {
		files[i] = root + t.fileSeparator + files[i]
	}
	for i := range dirs {
		dirs[i] = root + t.fileSeparator + dirs[i]
	}
	return
}

// GetConfigurations 获取配置类型和所在包名，再获取import
func (t *TypeScanner) GetConfigurations(filePath string) (types [][]string, imports []string, packageName string, aliasMap map[string]string) {
	types = make([][]string, 0)
	fileContent, err := t.getFileContentAsStringLines(filePath)
	if err != nil {
		return
	}
	indexes := t.findConfigurationsPosition(fileContent)
	configurations, aliasMap := t.getConfiguration(fileContent, indexes)
	packageName = t.getPackage(fileContent)
	imports = t.getImports(fileContent)
	return configurations, imports, packageName, aliasMap
}

func (t *TypeScanner) getFileContentAsStringLines(filePath string) ([]string, error) {
	var result []string
	by, err := ioutil.ReadFile(filePath)
	if err != nil {
		return nil, err
	}
	str := string(by)
	for _, lineStr := range strings.Split(str, "\n") {
		lineStr = strings.TrimSpace(lineStr)
		if lineStr == "" {
			continue
		}
		result = append(result, lineStr)
	}
	return result, nil
}

//判断本行是否是注释
func (t *TypeScanner) IsCodeAnnotation(str string) bool {
	if strings.TrimSpace(str)[:2] == "//" {
		return true
	}
	return false
}

//获取Configuration注解出现的位置
func (t *TypeScanner) findConfigurationsPosition(fileContent []string) []int {
	indexes := make([]int, 0)
	for i, line := range fileContent {
		if strings.Index(line, "//@Configuration") != -1 && !t.IsCodeAnnotation(fileContent[i+1]) {
			indexes = append(indexes, i)
		}
	}
	return indexes
}

func (t *TypeScanner) getConfiguration(fileContent []string, indexes []int) (types [][]string, aliasMap map[string]string) {
	types = make([][]string, 0)
	aliasMap = make(map[string]string)
	for _, index := range indexes {
		count := 0
		type_ := make([]string, 0)
		pos := index + 1
		typeAnnotation := fileContent[index]
		typeInfo := fileContent[pos]
		name := t.GetTypeName(typeInfo)
		alias := t.GetTypeAlias(typeAnnotation, name)
		aliasMap[name] = alias
		for ; ; pos++ {
			leftCount := strings.Count(fileContent[pos], "{")
			count += leftCount

			rightCount := strings.Count(fileContent[pos], "}")
			count -= rightCount

			if count == 0 {
				pos++
				break
			}
		}
		type_ = append(type_, fileContent[index+1:pos]...)
		types = append(types, type_)
	}
	return
}

func (t *TypeScanner) getPackage(fileContent []string) string {
	for i := range fileContent {
		if strings.Index(fileContent[i], "package") != -1 {
			return strings.TrimSpace(fileContent[i][len("package "):])
		}
	}
	panic("cannot find package name")
}

func (t *TypeScanner) getImports(fileContent []string) (imports []string) {
	imports = make([]string, 0)
	for i, line := range fileContent {
		if strings.Index(line, "import") != -1 {
			index := i
			left := -1
			for ; ; index++ {
				first := strings.Index(fileContent[index], "\"")
				end := strings.LastIndex(fileContent[index], "\"")
				l := strings.Index(fileContent[index], "(")
				if l != -1 {
					left = index
				}
				if first != -1 && end != -1 {
					item := fileContent[index][first+1 : end]
					imports = append(imports, item)
					if left == -1 {
						break
					}
				}
				if strings.Contains(fileContent[index], ")") {
					break
				}
			}
		}
	}
	return imports
}
