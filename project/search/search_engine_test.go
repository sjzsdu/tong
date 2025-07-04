package search

import (
	"os"
	"testing"

	"github.com/sjzsdu/tong/project"
	"github.com/stretchr/testify/assert"
)

// TestSearchEngine 测试搜索引擎的基本功能
func TestSearchEngine(t *testing.T) {
	// 使用共享项目
	goProject := project.GetSharedProject(t, "")
	proj := goProject.GetProject()

	// 创建搜索引擎
	engine := NewDefaultSearchEngine()
	assert.NotNil(t, engine)

	// 构建索引
	err := engine.BuildIndex(proj)
	assert.NoError(t, err)

	// 简单搜索
	results, err := engine.Search("main", SearchOptions{})
	assert.NoError(t, err)
	assert.NotEmpty(t, results, "应该找到包含'main'的结果")
}

// TestSearchEngineWithExampleProject 在示例项目上测试搜索引擎
func TestSearchEngineWithExampleProject(t *testing.T) {
	// 创建一个示例项目
	projectPath := project.CreateExampleGoProject(t)
	defer os.RemoveAll(projectPath) // 测试结束后清理

	// 创建一个新的项目实例，而不是使用共享实例
	options := project.DefaultWalkDirOptions()
	proj, err := project.BuildProjectTree(projectPath, options)
	assert.NoError(t, err)

	// 创建搜索引擎
	engine := NewDefaultSearchEngine()

	// 构建索引
	err = engine.BuildIndex(proj)
	assert.NoError(t, err)

	// 搜索已知存在的关键字
	searchTerms := []string{"func", "import", "Hello", "package"}

	for _, term := range searchTerms {
		results, err := engine.Search(term, SearchOptions{})
		assert.NoError(t, err)
		assert.NotEmpty(t, results, "应该找到包含'%s'的结果", term)
	}

	// 搜索不存在的关键字
	results, err := engine.Search("ThisStringDoesNotExistAnywhere123456789", SearchOptions{})
	assert.NoError(t, err)
	assert.Empty(t, results, "不应该找到不存在的关键字")
}

// TestSearchOptions 测试搜索选项
func TestSearchOptions(t *testing.T) {
	// 使用共享项目
	goProject := project.GetSharedProject(t, "")
	proj := goProject.GetProject()

	// 创建搜索引擎
	engine := NewDefaultSearchEngine()
	err := engine.BuildIndex(proj)
	assert.NoError(t, err)

	// 测试区分大小写
	caseSensitiveResults, err := engine.Search("MAIN", SearchOptions{CaseSensitive: true})
	assert.NoError(t, err)

	caseInsensitiveResults, err := engine.Search("MAIN", SearchOptions{CaseSensitive: false})
	assert.NoError(t, err)

	// 不区分大小写的搜索应该返回更多结果
	assert.GreaterOrEqual(t, len(caseInsensitiveResults), len(caseSensitiveResults),
		"不区分大小写的搜索应该找到至少与区分大小写相同数量的结果")

	// 测试文件类型限制
	goResults, err := engine.Search("func", SearchOptions{FileTypes: []string{"go"}})
	assert.NoError(t, err)

	mdResults, err := engine.Search("func", SearchOptions{FileTypes: []string{"md"}})
	assert.NoError(t, err)

	// Go文件中应该有更多"func"
	assert.GreaterOrEqual(t, len(goResults), len(mdResults),
		"Go文件中应该包含更多的'func'关键字")

	// 测试最大结果数
	limitedResults, err := engine.Search("the", SearchOptions{MaxResults: 5})
	assert.NoError(t, err)
	assert.LessOrEqual(t, len(limitedResults), 5, "结果数应该受到限制")
}

// TestRegexSearch 测试正则表达式搜索
func TestRegexSearch(t *testing.T) {
	// 使用共享项目
	goProject := project.GetSharedProject(t, "")
	proj := goProject.GetProject()

	// 创建搜索引擎
	engine := NewDefaultSearchEngine()
	err := engine.BuildIndex(proj)
	assert.NoError(t, err)

	// 使用正则表达式搜索
	regexResults, err := engine.Search("func\\s+\\w+\\s*\\(", SearchOptions{RegexMode: true})
	assert.NoError(t, err)
	assert.NotEmpty(t, regexResults, "应该找到匹配函数声明的正则表达式")

	// 无效的正则表达式
	_, err = engine.Search("[invalid", SearchOptions{RegexMode: true})
	assert.Error(t, err, "无效的正则表达式应该返回错误")
}

// TestWholeWordSearch 测试全词匹配
func TestWholeWordSearch(t *testing.T) {
	// 创建一个示例项目
	projectPath := project.CreateExampleGoProject(t)
	defer os.RemoveAll(projectPath) // 测试结束后清理

	// 使用示例项目创建 GoProject 实例
	goProject := project.GetSharedProject(t, projectPath)
	proj := goProject.GetProject()

	// 创建搜索引擎
	engine := NewDefaultSearchEngine()
	err := engine.BuildIndex(proj)
	assert.NoError(t, err)

	// 测试全词匹配
	wholeWordResults, err := engine.Search("func", SearchOptions{WholeWord: true})
	assert.NoError(t, err)

	// 测试部分匹配
	partialWordResults, err := engine.Search("func", SearchOptions{WholeWord: false})
	assert.NoError(t, err)

	// 全词匹配应该少于或等于部分匹配
	assert.LessOrEqual(t, len(wholeWordResults), len(partialWordResults),
		"全词匹配应该找到少于或等于部分匹配的结果数")
}
