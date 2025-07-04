package analyzer

import (
	"testing"

	"github.com/sjzsdu/tong/project"
	"github.com/stretchr/testify/assert"
)

// TestCodeAnalyzer 测试代码分析器
func TestCodeAnalyzer(t *testing.T) {
	// 使用共享项目进行测试
	goProject := project.GetSharedProject(t, "")
	proj := goProject.GetProject()

	// 创建代码分析器
	analyzer := NewDefaultCodeAnalyzer()
	assert.NotNil(t, analyzer)

	// 执行分析
	stats, err := analyzer.Analyze(proj)
	assert.NoError(t, err)
	assert.NotNil(t, stats)

	// 验证基本统计信息
	assert.Greater(t, stats.TotalFiles, 0, "应该有文件存在")
	assert.Greater(t, stats.TotalDirs, 0, "应该有目录存在")
	assert.Greater(t, stats.TotalLines, 0, "应该有代码行")
	assert.Greater(t, stats.TotalSize, int64(0), "应该有文件大小")

	// 验证语言统计
	assert.NotEmpty(t, stats.LanguageStats, "应该有语言统计")
	assert.NotEmpty(t, stats.FileTypeStats, "应该有文件类型统计")

	// 验证Go语言统计（因为我们的项目包含Go文件）
	goLines, hasGo := stats.LanguageStats["Go"]
	assert.True(t, hasGo, "应该包含Go语言统计")
	assert.Greater(t, goLines, 0, "Go代码行数应该大于0")

	// 验证文件类型统计
	goFiles, hasGoFiles := stats.FileTypeStats["go"]
	assert.True(t, hasGoFiles, "应该包含.go文件统计")
	assert.Greater(t, goFiles, 0, ".go文件数量应该大于0")
}

// TestCodeAnalyzerLanguageMapping 测试语言映射
func TestCodeAnalyzerLanguageMapping(t *testing.T) {
	analyzer := NewDefaultCodeAnalyzer()

	// 测试常见的语言映射
	expectedMappings := map[string]string{
		"go":   "Go",
		"py":   "Python",
		"js":   "JavaScript",
		"ts":   "TypeScript",
		"java": "Java",
		"md":   "Markdown",
		"json": "JSON",
	}

	for ext, expectedLang := range expectedMappings {
		actualLang, exists := analyzer.languageMap[ext]
		assert.True(t, exists, "扩展名 %s 应该存在映射", ext)
		assert.Equal(t, expectedLang, actualLang, "扩展名 %s 的语言映射不正确", ext)
	}
}

// TestCodeAnalyzerWithExampleProject 测试在示例项目上的代码分析
func TestCodeAnalyzerWithExampleProject(t *testing.T) {
	// 创建一个示例项目
	projectPath := project.CreateExampleGoProject(t)
	goProject := project.GetSharedProject(t, projectPath)
	proj := goProject.GetProject()

	// 创建代码分析器
	analyzer := NewDefaultCodeAnalyzer()

	// 执行分析
	stats, err := analyzer.Analyze(proj)
	assert.NoError(t, err)
	assert.NotNil(t, stats)

	// 验证示例项目的统计信息
	assert.Greater(t, stats.TotalFiles, 0)
	assert.Greater(t, stats.TotalDirs, 0)

	// 验证包含了 Go 代码
	_, hasGo := stats.LanguageStats["Go"]
	assert.True(t, hasGo, "示例项目应该包含Go代码")
}

// TestDependencyAnalyzer 测试依赖分析器
func TestDependencyAnalyzer(t *testing.T) {
	// 使用共享项目
	goProject := project.GetSharedProject(t, "")
	proj := goProject.GetProject()

	// 创建依赖分析器
	analyzer := NewDefaultDependencyAnalyzer()
	assert.NotNil(t, analyzer)

	// 执行依赖分析
	graph, err := analyzer.AnalyzeDependencies(proj)
	assert.NoError(t, err)
	assert.NotNil(t, graph)

	// 验证依赖图结构
	assert.NotNil(t, graph.Nodes, "依赖节点应该存在")
	assert.NotNil(t, graph.Edges, "依赖关系应该存在")

	// 如果项目包含Go模块，应该有一些依赖
	if len(graph.Nodes) > 0 {
		// 验证节点信息
		for name, node := range graph.Nodes {
			assert.NotEmpty(t, name, "依赖名称不应该为空")
			assert.NotNil(t, node, "依赖节点不应该为空")
			assert.NotEmpty(t, node.Name, "节点名称不应该为空")
			assert.NotEmpty(t, node.Type, "节点类型不应该为空")
		}
	}
}

// TestDependencyAnalyzerLanguageSupport 测试不同语言的依赖分析
func TestDependencyAnalyzerLanguageSupport(t *testing.T) {
	analyzer := NewDefaultDependencyAnalyzer()

	// 测试Go依赖分析器
	goAnalyzer := analyzer.languageAnalyzers[".go"]
	assert.NotNil(t, goAnalyzer, "应该支持Go语言依赖分析")

	// 测试JavaScript依赖分析器
	jsAnalyzer := analyzer.languageAnalyzers[".js"]
	assert.NotNil(t, jsAnalyzer, "应该支持JavaScript语言依赖分析")

	// 测试Python依赖分析器
	pyAnalyzer := analyzer.languageAnalyzers[".py"]
	assert.NotNil(t, pyAnalyzer, "应该支持Python语言依赖分析")
}

// TestGoLanguageDependencyAnalyzer 测试Go语言依赖分析器
func TestGoLanguageDependencyAnalyzer(t *testing.T) {
	analyzer := &GoDependencyAnalyzer{}

	// 测试Go代码依赖分析
	goCode := []byte(`package main

import "fmt"
import "os"
import "github.com/stretchr/testify/assert"

func main() {
	fmt.Println("Hello, World!")
}
`)

	nodes, edges, err := analyzer.Analyze(goCode, "main.go")
	assert.NoError(t, err)

	// 验证分析结果
	assert.NotEmpty(t, nodes, "应该识别出依赖节点")
	assert.NotEmpty(t, edges, "应该识别出依赖关系")

	// 验证标准库依赖
	foundFmt := false
	foundOs := false
	for _, node := range nodes {
		if node.Name == "fmt" {
			foundFmt = true
		}
		if node.Name == "os" {
			foundOs = true
		}
	}
	assert.True(t, foundFmt, "应该识别出fmt依赖")
	assert.True(t, foundOs, "应该识别出os依赖")
}

// TestJavaScriptLanguageDependencyAnalyzer 测试JavaScript语言依赖分析器
func TestJavaScriptLanguageDependencyAnalyzer(t *testing.T) {
	analyzer := &JSDependencyAnalyzer{}

	// 测试JavaScript代码依赖分析
	jsCode := []byte(`
const express = require('express');
const fs = require('fs');
import React from 'react';
import { Component } from 'react';

const app = express();
`)

	nodes, edges, err := analyzer.Analyze(jsCode, "app.js")
	assert.NoError(t, err)

	// 验证分析结果
	assert.NotEmpty(t, nodes, "应该识别出依赖节点")
	assert.NotEmpty(t, edges, "应该识别出依赖关系")

	// 验证依赖
	foundExpress := false
	foundReact := false
	for _, node := range nodes {
		if node.Name == "express" {
			foundExpress = true
		}
		if node.Name == "react" {
			foundReact = true
		}
	}
	assert.True(t, foundExpress, "应该识别出express依赖")
	assert.True(t, foundReact, "应该识别出react依赖")
}

// TestPythonLanguageDependencyAnalyzer 测试Python语言依赖分析器
func TestPythonLanguageDependencyAnalyzer(t *testing.T) {
	analyzer := &PythonDependencyAnalyzer{}

	// 测试Python代码依赖分析
	pyCode := []byte(`
import os
import sys
from flask import Flask, request
import numpy
import pandas

app = Flask(__name__)
`)

	nodes, edges, err := analyzer.Analyze(pyCode, "app.py")
	assert.NoError(t, err)

	// 验证分析结果
	assert.NotEmpty(t, nodes, "应该识别出依赖节点")
	assert.NotEmpty(t, edges, "应该识别出依赖关系")

	// 验证依赖
	foundFlask := false
	foundNumpy := false
	for _, node := range nodes {
		if node.Name == "flask" {
			foundFlask = true
		}
		if node.Name == "numpy" {
			foundNumpy = true
		}
	}
	assert.True(t, foundFlask, "应该识别出flask依赖")
	assert.True(t, foundNumpy, "应该识别出numpy依赖")
}

// TestDependencyGraphStructure 测试依赖图结构
func TestDependencyGraphStructure(t *testing.T) {
	// 创建测试依赖图
	graph := &DependencyGraph{
		Nodes: make(map[string]*DependencyNode),
		Edges: make(map[string][]string),
	}

	// 添加测试节点
	graph.Nodes["node1"] = &DependencyNode{
		Name:    "node1",
		Version: "1.0.0",
		Type:    "direct",
	}
	graph.Nodes["node2"] = &DependencyNode{
		Name:    "node2",
		Version: "2.0.0",
		Type:    "indirect",
	}

	// 添加测试边
	graph.Edges["node1"] = []string{"node2"}

	// 验证图结构
	assert.Len(t, graph.Nodes, 2, "应该有2个节点")
	assert.Len(t, graph.Edges, 1, "应该有1个边")

	// 验证节点属性
	node1 := graph.Nodes["node1"]
	assert.Equal(t, "node1", node1.Name)
	assert.Equal(t, "1.0.0", node1.Version)
	assert.Equal(t, "direct", node1.Type)

	// 验证边关系
	edges := graph.Edges["node1"]
	assert.Contains(t, edges, "node2", "node1应该依赖node2")
}
