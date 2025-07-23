package cmd

var (
	workDir         string
	extensions      []string
	outputFile      string
	excludePatterns []string
	repoURL         string
	skipGitIgnore   bool
	debugMode       bool
	configOptions   = map[string]string{
		"lang":     "Set language",
		"renderer": "Set llm response render type",
	}
	showAllConfigs bool
	configFile     string
)
