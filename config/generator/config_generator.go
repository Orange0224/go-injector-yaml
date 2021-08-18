package config_generator

import (
	"fmt"
	"github.com/orange0224/go-injector-yaml/config/utils"
	"io/ioutil"
	"os"
	"path/filepath"
	"runtime"
	"strings"
)

const (
	Template1 = `
case "{$type}":
		target := original.({$type})
		valueOfObject = reflect.ValueOf(&target)
		typeOfObject = reflect.TypeOf(&target)
`
	Template2 = `
case "{$type}":
					valueOfObject.Elem().FieldByName(field.Name).Set(reflect.ValueOf(result.({$type})))
`
	Header = `
func (l *Loader)mergeConfig(loadedConfig map[string]string, prefix, realType string, original interface{}, validator func(str string) bool) interface{} {
	var valueOfObject reflect.Value
	var typeOfObject reflect.Type

	switch realType {
case "ApplicationConfig":
		target := original.(ApplicationConfig)
		valueOfObject = reflect.ValueOf(&target)
		typeOfObject = reflect.TypeOf(&target)
`
	Content1 = `
}

	if typeOfObject.Elem().Kind() == reflect.Struct {
		for i := 0; i < typeOfObject.Elem().NumField(); i++ {
			field := typeOfObject.Elem().Field(i)
			fieldValue := valueOfObject.Elem().FieldByName(field.Name)
			key := field.Tag[6 : len(field.Tag)-1]
			if field.Type.Kind() == reflect.Struct {
				oldPrefix := prefix
				prefix += string(key) + "."
				result := l.mergeConfig(loadedConfig, prefix, field.Type.Name(), fieldValue.Interface(), validator)
				switch field.Type.Name() {
`
	Footer = `
}
				prefix = oldPrefix
			} else {
				if validator(loadedConfig[prefix+string(key)]) {
					valueOfObject.Elem().FieldByName(field.Name).SetString(loadedConfig[prefix+string(key)])
				}
			}
		}
	}
	return valueOfObject.Elem().Interface()
}
`
	ConfigLoaderTemplate = `package config

import (
	"fmt"
	"go-injector-yaml/config/utils"
	"io/ioutil"
	"net/http"
	"os"
	"reflect"
	"strings"

	"gopkg.in/yaml.v2"
)

type Loader struct {
	External     bool
	Cloud        bool
	CloudAddress string
	ConfigPath   string
}

func (l *Loader) configValidator() {
	if l.Cloud {
		if utils.IsBlank(l.CloudAddress) {
			panic("cloud config path cannot be empty if cloud is enabled")
		}
	}
}
func (l *Loader) Begin() {
	l.configValidator()
	l.initDefaultConfig()
	if l.External {
		if utils.NotBlank(l.ConfigPath) {
			l.initConfigFromFile()
		}
		if l.Cloud {
			l.initConfigFromCloud()
		}
		l.initConfigFromArgs()
	}
	config = l.register(applicationConfig)
	configStr = l.getStringSet(config)
}

//@DefaultConfigGenerate
//@AutoExecuteGenerate

func (l *Loader) loadConfigFromFile() (int, []byte, error) {
	//file, err := os.Open("/home/lxs/GoProject/storage/part9/apiServer/src/application.yaml")
	file, err := os.Open(l.ConfigPath)
	if err != nil {
		return 0, nil, err
	}
	defer file.Close()
	fileInfo, err := file.Stat()
	if err != nil {
		return 0, nil, err
	}
	fileSize := fileInfo.Size()
	buffer := make([]byte, fileSize)
	bytes, err := file.Read(buffer)
	if err != nil {
		return 0, nil, err
	}
	return bytes, buffer, nil
}
func (l *Loader) initConfigFromFile() {
	_, bytes, err := l.loadConfigFromFile()
	if err != nil {
		return
	}
	l.loadConfigFromBytes(bytes)
}
func (l *Loader) loadConfigFromBytes(bytes []byte) {
	var fileConfig ApplicationConfig
	yaml.Unmarshal(bytes, &fileConfig)
	configMap := l.register(fileConfig)
	configStringMap := l.getStringSet(configMap)
	configLocal := l.mergeConfig(configStringMap, "", "ApplicationConfig", applicationConfig, utils.NotBlank)
	applicationConfig = configLocal.(ApplicationConfig)
}
func (l *Loader) initConfigFromCloud() {
	url := l.CloudAddress
	fmt.Println("get config from cloud:")
	response, err := http.Get(url)
	if err != nil {
		return
	}
	defer response.Body.Close()
	buffer, err := ioutil.ReadAll(response.Body)
	if err != nil {
		return
	}
	l.loadConfigFromBytes(buffer)
	fmt.Println("load cloud config success!")
}
func (l *Loader) initConfigFromArgs() {
	keyStringSet := make(map[string]string, 0)
	for _, command := range os.Args {
		if command[0] == '-' {
			key := command[1:strings.Index(command, "=")]
			value := command[strings.Index(command, "=")+1:]
			keyStringSet[key] = value
		}
	}
	for k, v := range keyStringSet {
		fmt.Println(k, v)
	}
	configLocal := l.mergeConfig(keyStringSet, "", "ApplicationConfig", applicationConfig, utils.NotBlank)
	applicationConfig = configLocal.(ApplicationConfig)
}
func (l *Loader) register(object interface{}) map[string]interface{} {
	keySet := make(map[string]interface{})
	typeOfInterface := reflect.TypeOf(object)
	valueOfInterface := reflect.ValueOf(object)
	if typeOfInterface.Kind() == reflect.Struct {
		for i := 0; i < typeOfInterface.NumField(); i++ {
			field := typeOfInterface.Field(i)
			key := field.Tag[6 : len(field.Tag)-1]
			value := valueOfInterface.FieldByName(field.Name)
			keySet[string(key)] = value.Interface()
			fieldType := reflect.TypeOf(field)
			if fieldType.Kind() == reflect.Struct {
				prefix := string(key) + "."
				subKeySet := l.register(value.Interface())
				for k, v := range subKeySet {
					keySet[prefix+k] = v
				}
			}
		}
	} else {
		return nil
	}
	return keySet
}
func (l *Loader) getStringSet(keySet map[string]interface{}) map[string]string {
	stringSet := make(map[string]string)
	for k, v := range keySet {
		if reflect.TypeOf(v).Kind() == reflect.String {
			stringSet[k] = v.(string)
		}
	}
	return stringSet
}

var config map[string]interface{}
var configStr map[string]string

//@MergeConfigGenerate

func GetString(key string) string {
	return configStr[key]
}
func GetConfig(key string) interface{} {
	return config[key]
}
`
)

type Generator struct {
	ConfigDir     string
	aliasMap      map[string]string
	fileSeparator string
	typeAlias     map[string]string
}

func (g *Generator) checkConfig() {
	if utils.IsBlank(g.ConfigDir) {
		panic("cannot scan empty path")
	}
}

func (g *Generator) initVariable() {
	if runtime.GOOS == "windows" {
		g.fileSeparator = "\\"
	} else {
		g.fileSeparator = "/"
	}
}

func (g *Generator) Begin() {
	g.initVariable()
	scanFiles := g.getScanFiles(g.ConfigDir)
	scanTypes, defaultConfig, autoExecute := g.getScanTypes(scanFiles)
	g.generateFunc(scanTypes)
	method := g.generateFunc(scanTypes)
	configs := g.generateConfigs(defaultConfig)
	executes := g.generateExecute(autoExecute)
	g.writeResultToFile(method, configs, executes, g.ConfigDir)
}

func (g *Generator) getScanFiles(configPath string) []string {
	fileInfoList, err := filepath.Glob(filepath.Join(configPath, "*.go"))
	if err != nil {
		return nil
	}
	files := make([]string, 0)
	for _, file := range fileInfoList {
		filename := file[strings.LastIndex(file, g.fileSeparator)+1:]
		if filename == "config_loader.go" {
			continue
		}
		files = append(files, file)
	}
	return files
}

func (g *Generator) getFileContentAsStringLines(filePath string) ([]string, error) {
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

func (g *Generator) getStructType(lines []string) []string {
	types := make([]string, 0)
	for _, line := range lines {
		typeIndex := strings.Index(string(line), "type ")
		structIndex := strings.Index(string(line), "struct ")
		if typeIndex != -1 && structIndex != -1 && typeIndex < structIndex {
			types = append(types, strings.TrimSpace(line[typeIndex+len("type "):structIndex]))
		}
	}
	return types
}

func (g *Generator) getScanTypes(files []string) (types []string, defaultConfig map[string]string, autoExecute []string) {
	types = make([]string, 0)
	defaultConfig = make(map[string]string, 0)
	autoExecute = make([]string, 0)
	for _, file := range files {
		content, _ := g.getFileContentAsStringLines(file)
		ty := g.getStructType(content)
		config := g.getDefaultConfig(content)
		execute := g.getAutoExecute(content)
		indexes := findConfigurationsPosition(content)
		_, aliasMap := getConfiguration(content, indexes)

		types = append(types, ty...)
		autoExecute = append(autoExecute, execute...)
		for k, v := range config {
			defaultConfig[k] = v
		}
		if g.aliasMap == nil {
			g.aliasMap = make(map[string]string)
		}
		for k, v := range aliasMap {
			g.aliasMap[k] = v
		}
	}
	return types, defaultConfig, autoExecute
}

func (g *Generator) getDefaultConfig(lines []string) map[string]string {
	indexes := make([]int, 0)
	configs := make(map[string]string)
	for i := range lines {
		line := lines[i]
		if strings.Index(line, "//@DefaultConfig") != -1 {
			indexes = append(indexes, i)
		}
	}
	for _, i := range indexes {
		line := lines[i+1]
		split := strings.Split(line, " ")
		if len(split) >= 3 {
			configs[split[2]] = split[1]
		}
	}
	return configs
}

func (g *Generator) getAutoExecute(lines []string) []string {
	indexes := make([]int, 0)
	methods := make([]string, 0)
	for i := range lines {
		line := lines[i]
		if strings.Index(line, "//@AutoExecute") != -1 {
			indexes = append(indexes, i)
		}
	}
	for _, i := range indexes {
		line := lines[i+1]
		split := strings.Split(line, " ")
		if len(split) >= 2 {
			methods = append(methods, split[1])
		}
	}
	return methods
}

func (g *Generator) getTemplate1(name string) string {
	return strings.ReplaceAll(Template1, "{$type}", name)
}

func (g *Generator) getTemplate2(name string) string {
	return strings.ReplaceAll(Template2, "{$type}", name)
}

func (g *Generator) generateFunc(types []string) string {
	template1 := ""
	template2 := ""
	for _, ty := range types {
		template1 += g.getTemplate1(ty)
		template2 += g.getTemplate2(ty)

	}
	return Header + template1 + Content1 + template2 + Footer
}

func (g *Generator) generateConfigs(config map[string]string) string {
	header := `
func (l *Loader) initDefaultConfig() {
`
	methods := ""
	for typ, method := range config {
		methods += "applicationConfig." + strings.ToUpper(g.aliasMap[typ][0:1]) + g.aliasMap[typ][1:] + "=" + method + "\n"
	}
	footer := `}
`
	return header + methods + footer
}
func IsCodeAnnotation(str string) bool {
	if strings.TrimSpace(str)[:2] == "//" {
		return true
	}
	return false
}
func GetTypeName(line string) string {
	typeIndex := strings.Index(line, "type ")
	structIndex := strings.Index(line, "struct ")
	if typeIndex != -1 && structIndex != -1 && typeIndex < structIndex {
		return strings.TrimSpace(line[typeIndex+len("type ") : structIndex])
	}
	return ""
}

func GetTypeAlias(line, typeName string) string {
	index := strings.Index(line, "@Alias=")
	if index == -1 {
		return strings.ToLower(typeName[0:1]) + typeName[1:]
	}
	split := strings.Split(line[index+len("@Alias="):], " ")
	return split[0]
}

func getConfiguration(fileContent []string, indexes []int) (types [][]string, aliasMap map[string]string) {
	types = make([][]string, 0)
	aliasMap = make(map[string]string)
	for _, index := range indexes {
		count := 0
		type_ := make([]string, 0)
		pos := index + 1
		typeAnnotation := fileContent[index]
		typeInfo := fileContent[pos]
		name := GetTypeName(typeInfo)
		alias := GetTypeAlias(typeAnnotation, name)
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

//获取Configuration注解出现的位置
func findConfigurationsPosition(fileContent []string) []int {
	indexes := make([]int, 0)
	for i, line := range fileContent {
		if strings.Index(line, "//@Configuration") != -1 && !IsCodeAnnotation(fileContent[i+1]) {
			indexes = append(indexes, i)
		}
	}
	return indexes
}

func (g *Generator) generateExecute(executes []string) string {
	header := `
func (l *Loader) autoExecute() {
`
	methods := ""
	for _, method := range executes {
		methods += method + "\n"
	}
	footer := `}
`
	return header + methods + footer
}

func (g *Generator) writeResultToFile(method, config, execute, configPath string) {
	templateContent := strings.Split(ConfigLoaderTemplate, "\n")
	methodIndex, configIndex, executeIndex := -1, -1, -1
	for i := range templateContent {
		line := templateContent[i]
		mIndex := strings.Index(line, "//@MergeConfigGenerate")
		cIndex := strings.Index(line, "//@DefaultConfigGenerate")
		eIndex := strings.Index(line, "//@AutoExecuteGenerate")
		if mIndex != -1 {
			if methodIndex != -1 {
				panic("error occurred when scan @MergeConfigGenerate:Multiple instances detected")
			}
			methodIndex = i
		}
		if cIndex != -1 {
			if configIndex != -1 {
				panic("error occurred when scan @DefaultConfigGenerate:Multiple instances detected")
			}
			configIndex = i
		}
		if eIndex != -1 {
			if executeIndex != -1 {
				panic("error occurred when scan @AutoExecuteGenerate:Multiple instances detected")
			}
			executeIndex = i
		}
	}
	if methodIndex == -1 || configIndex == -1 || executeIndex == -1 {
		panic(fmt.Errorf("cannot find enough instance:method:%d,config:%d,execute:%d", methodIndex, configIndex, executeIndex))
	}

	if methodIndex == configIndex || methodIndex == executeIndex || configIndex == executeIndex {
		panic(fmt.Errorf("instance conflict:method:%d,config:%d,execute:%d", methodIndex, configIndex, executeIndex))
	}

	writeFile := make([]string, 0)
	for i, line := range templateContent {
		writeFile = append(writeFile, line)
		if i == methodIndex {
			writeFile = append(writeFile, method)
		}
		if i == configIndex {
			writeFile = append(writeFile, config)
		}
		if i == executeIndex {
			writeFile = append(writeFile, execute)
		}
	}
	file, err := os.Create(configPath + "/config_loader.go")
	if err != nil {
		fmt.Println(err)
		return
	}

	defer file.Close()
	for _, line := range writeFile {
		file.Write([]byte(line))
		file.Write([]byte("\n"))
	}
}
