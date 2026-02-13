package cmd

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/sjzsdu/tong/config"
	"github.com/sjzsdu/tong/dsync"
	"github.com/sjzsdu/tong/lang"
	"github.com/spf13/cobra"
)

var (
	dsyncFullDir    string
	dsyncCurrentDir string
	dsyncConfigFile string
)

var dsyncCmd = &cobra.Command{
	Use:   "dsync",
	Short: lang.T("Directory sync commands"),
	Long:  lang.T("Manage directory synchronization between source and target"),
	Run:   handleDsyncCommand,
}

var dsyncFullCmd = &cobra.Command{
	Use:   "full",
	Short: lang.T("Show source directory"),
	Long:  lang.T("Show source directory structure"),
	Run:   handleDsyncFullCommand,
}

var dsyncFullLevel int
var dsyncCurrentLevel int

var dsyncCurrentCmd = &cobra.Command{
	Use:   "current",
	Short: lang.T("Show target directory"),
	Long:  lang.T("Show target directory structure"),
	Run:   handleDsyncCurrentCommand,
}

var dsyncSyncCmd = &cobra.Command{
	Use:   "sync [keys...]",
	Short: lang.T("Sync keys from source to target"),
	Long:  lang.T("Sync specified keys from source DirMap to target DirMap, then write back"),
	Run:   handleDsyncSyncCommand,
}

var dsyncRemoveCmd = &cobra.Command{
	Use:   "remove [keys...]",
	Short: lang.T("Remove keys from target"),
	Long:  lang.T("Remove specified keys from target DirMap, then write back"),
	Run:   handleDsyncRemoveCommand,
}

var dsyncConfigCmd = &cobra.Command{
	Use:   "config",
	Short: lang.T("Show or set dsync paths"),
	Long:  lang.T("Show or set directory sync configuration paths"),
	Run:   handleDsyncConfigCommand,
}

func init() {
	rootCmd.AddCommand(dsyncCmd)
	dsyncCmd.AddCommand(dsyncFullCmd)
	dsyncCmd.AddCommand(dsyncCurrentCmd)
	dsyncCmd.AddCommand(dsyncSyncCmd)
	dsyncCmd.AddCommand(dsyncRemoveCmd)
	dsyncCmd.AddCommand(dsyncConfigCmd)

	home, _ := os.UserHomeDir()
	defaultFullDir := filepath.Join(home, ".config", "opencode-full")

	dsyncCmd.PersistentFlags().StringVar(&dsyncFullDir, "full-dir", "", lang.T("Source directory path"))
	dsyncCmd.PersistentFlags().StringVar(&dsyncCurrentDir, "current-dir", "", lang.T("Target directory path"))
	dsyncCmd.PersistentFlags().StringVar(&dsyncConfigFile, "config-file", "", lang.T("Config file name"))

	dsyncConfigCmd.Flags().String("set-full-dir", "", lang.T("Set source directory path"))
	dsyncConfigCmd.Flags().String("set-current-dir", "", lang.T("Set target directory path"))
	dsyncConfigCmd.Flags().String("set-config-file", "", lang.T("Set config file name"))

	dsyncFullCmd.Flags().IntVarP(&dsyncFullLevel, "level", "l", 0, lang.T("Show items up to this depth level (0 for all)"))
	dsyncCurrentCmd.Flags().IntVarP(&dsyncCurrentLevel, "level", "l", 0, lang.T("Show items up to this depth level (0 for all)"))

	_ = config.LoadConfig()

	if dsyncFullDir == "" {
		dsyncFullDir = config.GetConfigWithDefault(config.KeyDsyncFullDir, defaultFullDir)
	}
	if dsyncCurrentDir == "" {
		dsyncCurrentDir = config.GetConfigWithDefault(config.KeyDsyncCurrentDir, ".opencode")
	}
	if dsyncConfigFile == "" {
		dsyncConfigFile = config.GetConfigWithDefault(config.KeyDsyncConfigFile, "opencode.json")
	}
}

func getFullConfigDir() string {
	home, _ := os.UserHomeDir()
	defaultFullDir := filepath.Join(home, ".config", "opencode-full")
	return config.GetConfigWithDefault(config.KeyDsyncFullDir, defaultFullDir)
}

func getCurrentConfigDir() string {
	return config.GetConfigWithDefault(config.KeyDsyncCurrentDir, ".opencode")
}

func getConfigFile() string {
	return config.GetConfigWithDefault(config.KeyDsyncConfigFile, "opencode.json")
}

func handleDsyncCommand(cmd *cobra.Command, args []string) {
	cmd.Help()
}

// --- full 命令：展示源目录的 DirData ---

func handleDsyncFullCommand(cmd *cobra.Command, args []string) {
	fullDir := getFullConfigDir()

	if !fileExists(fullDir) {
		fmt.Println(lang.T("Source directory not found."))
		fmt.Printf(lang.T("Path: %s\n"), fullDir)
		return
	}

	dd, err := dsync.BuildDirData(fullDir, getConfigFile())
	if err != nil {
		fmt.Printf(lang.T("Error reading directory: %v\n"), err)
		return
	}

	fmt.Printf(lang.T("Source directory (%s):\n"), fullDir)
	fmt.Println()
	fmt.Print(dsync.DisplayDirMapWithDepth(dd.Data, dsyncFullLevel))
}

// --- current 命令：展示目标目录的 DirData ---

func handleDsyncCurrentCommand(cmd *cobra.Command, args []string) {
	currentDir := getCurrentConfigDir()

	if !fileExists(currentDir) {
		fmt.Println(lang.T("Target directory not found."))
		fmt.Printf(lang.T("Path: %s\n"), currentDir)
		return
	}

	dd, err := dsync.BuildDirData(currentDir, getConfigFile())
	if err != nil {
		fmt.Printf(lang.T("Error reading directory: %v\n"), err)
		return
	}

	fmt.Printf(lang.T("Target directory (%s):\n"), currentDir)
	fmt.Println()
	fmt.Print(dsync.DisplayDirMapWithDepth(dd.Data, dsyncCurrentLevel))
}

// --- sync 命令：从源 DirData 同步 key 到目标 DirData，然后写回 ---

func handleDsyncSyncCommand(cmd *cobra.Command, args []string) {
	fullDir := getFullConfigDir()
	currentDir := getCurrentConfigDir()
	configFile := getConfigFile()

	if !fileExists(fullDir) {
		fmt.Println(lang.T("Source directory not found."))
		fmt.Printf(lang.T("Path: %s\n"), fullDir)
		return
	}

	// 构建源 DirData
	source, err := dsync.BuildDirData(fullDir, configFile)
	if err != nil {
		fmt.Printf(lang.T("Error reading source directory: %v\n"), err)
		return
	}

	// 构建目标 DirData（如果目录不存在则创建空的）
	var target *dsync.DirData
	if fileExists(currentDir) {
		target, err = dsync.BuildDirData(currentDir, configFile)
		if err != nil {
			fmt.Printf(lang.T("Error reading target directory: %v\n"), err)
			return
		}
	} else {
		target = dsync.NewEmptyDirData(configFile)
	}

	if len(args) == 0 {
		// 全量同步：用源覆盖目标
		fmt.Println(lang.T("Syncing entire source to target..."))
		target.Data = source.Data
		target.DirKeys = source.DirKeys
		target.FileKeys = source.FileKeys
	} else {
		// 增量同步：同步指定 key
		fmt.Println(lang.T("Syncing specified keys to target..."))
		for _, key := range args {
			if err := target.SyncFrom(source, key); err != nil {
				fmt.Printf(lang.T("Error syncing key %s: %v\n"), key, err)
				continue
			}
			fmt.Printf(lang.T("Synced: %s\n"), key)
		}
	}

	// 确保 $schema 始终存在于目标中
	if schema, ok := source.Data["$schema"]; ok {
		target.Data["$schema"] = schema
	}

	// 写回目标目录
	if err := target.WriteTo(currentDir); err != nil {
		fmt.Printf(lang.T("Error writing target directory: %v\n"), err)
		return
	}

	fmt.Println(lang.T("Sync completed!"))
}

// --- remove 命令：从目标 DirData 删除 key，然后写回 ---

func handleDsyncRemoveCommand(cmd *cobra.Command, args []string) {
	currentDir := getCurrentConfigDir()
	configFile := getConfigFile()

	if !fileExists(currentDir) {
		fmt.Println(lang.T("Target directory not found."))
		fmt.Printf(lang.T("Path: %s\n"), currentDir)
		return
	}

	if len(args) == 0 {
		// 删除整个目标目录
		fmt.Println(lang.T("Removing entire target directory..."))
		if err := os.RemoveAll(currentDir); err != nil {
			fmt.Printf(lang.T("Error removing directory: %v\n"), err)
			return
		}
		fmt.Println(lang.T("Remove completed!"))
		return
	}

	// 构建目标 DirData
	target, err := dsync.BuildDirData(currentDir, configFile)
	if err != nil {
		fmt.Printf(lang.T("Error reading target directory: %v\n"), err)
		return
	}

	fmt.Println(lang.T("Removing specified keys from target..."))
	for _, key := range args {
		if target.DeleteValue(key) {
			fmt.Printf(lang.T("Removed: %s\n"), key)
		} else {
			fmt.Printf(lang.T("Key not found: %s\n"), key)
		}
	}

	// 先清空目标目录，再重新写入
	if err := os.RemoveAll(currentDir); err != nil {
		fmt.Printf(lang.T("Error removing directory: %v\n"), err)
		return
	}
	if err := target.WriteTo(currentDir); err != nil {
		fmt.Printf(lang.T("Error writing target directory: %v\n"), err)
		return
	}

	fmt.Println(lang.T("Remove completed!"))
}

// --- config 命令 ---

func handleDsyncConfigCommand(cmd *cobra.Command, args []string) {
	setFullDir, _ := cmd.Flags().GetString("set-full-dir")
	setCurrentDir, _ := cmd.Flags().GetString("set-current-dir")
	setConfigFile, _ := cmd.Flags().GetString("set-config-file")

	needSave := false

	if setFullDir != "" {
		config.SetConfig(config.KeyDsyncFullDir, setFullDir)
		fmt.Printf(lang.T("Source directory set to: %s\n"), setFullDir)
		needSave = true
	}
	if setCurrentDir != "" {
		config.SetConfig(config.KeyDsyncCurrentDir, setCurrentDir)
		fmt.Printf(lang.T("Target directory set to: %s\n"), setCurrentDir)
		needSave = true
	}
	if setConfigFile != "" {
		config.SetConfig(config.KeyDsyncConfigFile, setConfigFile)
		fmt.Printf(lang.T("Config file set to: %s\n"), setConfigFile)
		needSave = true
	}

	if needSave {
		if err := config.SaveConfig(); err != nil {
			fmt.Printf(lang.T("Error saving config: %v\n"), err)
			return
		}
		fmt.Println(lang.T("Configuration saved successfully!"))
		return
	}

	fmt.Println(lang.T("Current dsync configuration paths:"))
	fmt.Printf(lang.T("  Source directory: %s\n"), getFullConfigDir())
	fmt.Printf(lang.T("  Target directory: %s\n"), getCurrentConfigDir())
	fmt.Printf(lang.T("  Config file: %s\n"), getConfigFile())
	fmt.Println()
	fmt.Println(lang.T("Use --set-full-dir, --set-current-dir, --set-config-file to modify paths."))
}

func fileExists(path string) bool {
	_, err := os.Stat(path)
	return !os.IsNotExist(err)
}
