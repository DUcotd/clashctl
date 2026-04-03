package cmd

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/spf13/cobra"

	"clashctl/internal/app"
	"clashctl/internal/config"
	"clashctl/internal/core"
	"clashctl/internal/system"

	"gopkg.in/yaml.v3"
)

var (
	backupJSON  bool
	restoreJSON bool
)

type backupRunItem struct {
	Kind    string `json:"kind"`
	Source  string `json:"source"`
	Path    string `json:"path,omitempty"`
	Found   bool   `json:"found"`
	Created bool   `json:"created"`
	Warning string `json:"warning,omitempty"`
}

type backupRunReport struct {
	BackupDir string          `json:"backup_dir"`
	Items     []backupRunItem `json:"items"`
}

type backupListEntry struct {
	Name       string    `json:"name"`
	Path       string    `json:"path"`
	ModifiedAt time.Time `json:"modified_at"`
	SizeBytes  int64     `json:"size_bytes"`
	SizeHuman  string    `json:"size_human"`
}

type backupListReport struct {
	BackupDir string            `json:"backup_dir"`
	Entries   []backupListEntry `json:"entries"`
}

type restoreRunReport struct {
	BackupName     string   `json:"backup_name"`
	BackupPath     string   `json:"backup_path"`
	TargetPath     string   `json:"target_path"`
	PreviousBackup string   `json:"previous_backup,omitempty"`
	Restored       bool     `json:"restored"`
	Warnings       []string `json:"warnings,omitempty"`
	TargetKind     string   `json:"target_kind"`
}

var backupCmd = &cobra.Command{
	Use:   "backup",
	Short: "备份当前配置",
	Long:  `备份 Mihomo 配置文件到 ~/.config/clashctl/backup/ 目录。`,
	RunE:  runBackup,
}

var backupRestoreCmd = &cobra.Command{
	Use:   "restore [version]",
	Short: "恢复历史配置",
	Long:  `从备份中恢复配置文件。不指定版本则列出可用备份。`,
	Args:  cobra.MaximumNArgs(1),
	RunE:  runRestore,
}

var backupListCmd = &cobra.Command{
	Use:   "list",
	Short: "列出所有备份",
	RunE:  runBackupList,
}

func init() {
	backupCmd.PersistentFlags().BoolVar(&backupJSON, "json", false, "以 JSON 输出备份结果")
	backupRestoreCmd.Flags().BoolVar(&restoreJSON, "json", false, "以 JSON 输出恢复结果")
	backupCmd.AddCommand(backupListCmd)
	backupCmd.AddCommand(backupRestoreCmd)
	rootCmd.AddCommand(backupCmd)
}

// BackupDir returns the backup directory path.
func BackupDir() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("无法获取用户主目录: %w", err)
	}
	return backupDirForHome(home), nil
}

func backupDirForHome(home string) string {
	return filepath.Join(home, ".config", "clashctl", "backup")
}

func runBackup(cmd *cobra.Command, args []string) error {
	cfg, err := loadAppConfig()
	if err != nil {
		return err
	}

	backupDir, err := BackupDir()
	if err != nil {
		return err
	}

	if err := os.MkdirAll(backupDir, 0755); err != nil {
		return fmt.Errorf("创建备份目录失败: %w", err)
	}

	report, err := createBackupReport(cfg, backupDir, time.Now())
	if err != nil {
		return err
	}
	if backupJSON {
		return writeJSON(report)
	}
	printBackupReport(report)
	return nil
}

func createBackupReport(cfg *core.AppConfig, backupDir string, now time.Time) (*backupRunReport, error) {
	report := &backupRunReport{BackupDir: backupDir}
	timestamp := now.Format("20060102-150405")

	mihomoPath := mihomoConfigPath(cfg)
	mihomoItem := backupRunItem{Kind: "mihomo", Source: mihomoPath}
	if _, err := os.Stat(mihomoPath); err == nil {
		backupPath := filepath.Join(backupDir, fmt.Sprintf("config-%s.yaml", timestamp))
		if err := copyBackupFile(mihomoPath, backupPath); err != nil {
			return nil, fmt.Errorf("备份 Mihomo 配置失败: %w", err)
		}
		mihomoItem.Found = true
		mihomoItem.Created = true
		mihomoItem.Path = backupPath
	} else if os.IsNotExist(err) {
		mihomoItem.Warning = "未找到 Mihomo 配置文件"
	} else {
		return nil, fmt.Errorf("检查 Mihomo 配置失败: %w", err)
	}
	report.Items = append(report.Items, mihomoItem)

	clashctlConfigPath, err := app.ConfigPath()
	clashctlItem := backupRunItem{Kind: "clashctl"}
	if err != nil {
		clashctlItem.Warning = fmt.Sprintf("获取 clashctl 配置路径失败: %v", err)
		report.Items = append(report.Items, clashctlItem)
		return report, nil
	}
	clashctlItem.Source = clashctlConfigPath
	if _, err := os.Stat(clashctlConfigPath); err == nil {
		backupPath := filepath.Join(backupDir, fmt.Sprintf("clashctl-%s.yaml", timestamp))
		if err := copyBackupFile(clashctlConfigPath, backupPath); err != nil {
			clashctlItem.Warning = fmt.Sprintf("备份 clashctl 配置失败: %v", err)
		} else {
			clashctlItem.Found = true
			clashctlItem.Created = true
			clashctlItem.Path = backupPath
		}
	} else if os.IsNotExist(err) {
		clashctlItem.Warning = "未找到 clashctl 配置文件"
	} else {
		clashctlItem.Warning = fmt.Sprintf("检查 clashctl 配置失败: %v", err)
	}
	report.Items = append(report.Items, clashctlItem)

	return report, nil
}

func runBackupList(cmd *cobra.Command, args []string) error {
	backupDir, err := BackupDir()
	if err != nil {
		return err
	}

	entries, err := listBackupEntries(backupDir)
	if err != nil {
		return err
	}
	report := &backupListReport{BackupDir: backupDir, Entries: entries}
	if backupJSON {
		return writeJSON(report)
	}
	printBackupList(report)
	return nil
}

func listBackupEntries(backupDir string) ([]backupListEntry, error) {
	if _, err := os.Stat(backupDir); os.IsNotExist(err) {
		return nil, nil
	} else if err != nil {
		return nil, fmt.Errorf("读取备份目录失败: %w", err)
	}

	entries, err := os.ReadDir(backupDir)
	if err != nil {
		return nil, fmt.Errorf("读取备份目录失败: %w", err)
	}

	var backups []backupListEntry
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		info, err := entry.Info()
		if err != nil {
			continue
		}
		backups = append(backups, backupListEntry{
			Name:       entry.Name(),
			Path:       filepath.Join(backupDir, entry.Name()),
			ModifiedAt: info.ModTime(),
			SizeBytes:  info.Size(),
			SizeHuman:  formatSize(info.Size()),
		})
	}

	sort.Slice(backups, func(i, j int) bool {
		return backups[i].ModifiedAt.After(backups[j].ModifiedAt)
	})

	return backups, nil
}

func printBackupReport(report *backupRunReport) {
	for _, item := range report.Items {
		label := item.Kind
		if item.Kind == "mihomo" {
			label = "配置"
		} else if item.Kind == "clashctl" {
			label = "clashctl 配置"
		}
		switch {
		case item.Created:
			fmt.Printf("✅ %s已备份到: %s\n", label, item.Path)
		case item.Warning != "":
			fmt.Printf("⚠️  %s\n", item.Warning)
		}
	}
	if len(report.Items) == 0 {
		fmt.Println("没有可备份的配置")
	}
}

func printBackupList(report *backupListReport) {
	if len(report.Entries) == 0 {
		fmt.Println("没有备份文件")
		return
	}

	fmt.Println("📦 可用备份:")
	fmt.Println()
	for _, entry := range report.Entries {
		fmt.Printf("  %-40s  %s  %s\n", entry.Name, entry.ModifiedAt.Format("2006-01-02 15:04:05"), entry.SizeHuman)
	}
	fmt.Println()
	fmt.Println("使用 'clashctl backup restore <文件名>' 恢复配置")
}

func runRestore(cmd *cobra.Command, args []string) error {
	backupDir, err := BackupDir()
	if err != nil {
		return err
	}

	// If no args, list available backups
	if len(args) == 0 {
		if restoreJSON {
			entries, err := listBackupEntries(backupDir)
			if err != nil {
				return err
			}
			return writeJSON(&backupListReport{BackupDir: backupDir, Entries: entries})
		}
		return runBackupList(cmd, args)
	}

	backupName := args[0]
	backupPath, err := resolveBackupPath(backupDir, backupName)
	if err != nil {
		return err
	}

	// Check if backup exists
	if _, err := os.Stat(backupPath); os.IsNotExist(err) {
		return fmt.Errorf("备份文件不存在: %s", backupName)
	}

	// Determine target based on backup name
	cfg, err := loadAppConfig()
	if err != nil {
		return err
	}

	var targetPath string
	if strings.HasPrefix(backupName, "clashctl-") {
		targetPath, err = app.ConfigPath()
		if err != nil {
			return err
		}
	} else {
		targetPath = filepath.Join(cfg.ConfigDir, "config.yaml")
	}

	// Read backup
	data, err := config.ReadConfigWithLimit(backupPath)
	if err != nil {
		return fmt.Errorf("读取备份失败: %w", err)
	}

	// Backup current config before restoring
	report := &restoreRunReport{
		BackupName: backupName,
		BackupPath: backupPath,
		TargetPath: targetPath,
		Restored:   false,
		TargetKind: "mihomo",
	}
	if strings.HasPrefix(backupName, "clashctl-") {
		report.TargetKind = "clashctl"
	}
	if err := validateRestoreData(data, report.TargetKind, backupPath); err != nil {
		return err
	}
	if _, err := os.Stat(targetPath); err == nil {
		previousBackup, err := config.BackupFile(targetPath)
		if err != nil {
			report.Warnings = append(report.Warnings, fmt.Sprintf("备份当前配置失败: %v", err))
		} else {
			report.PreviousBackup = previousBackup
		}
	}

	// Restore
	if err := config.WriteConfig(targetPath, data); err != nil {
		return fmt.Errorf("恢复配置失败: %w", err)
	}
	report.Restored = true
	if restoreJSON {
		return writeJSON(report)
	}
	for _, warning := range report.Warnings {
		fmt.Printf("⚠️  %s\n", warning)
	}

	fmt.Printf("✅ 配置已从 %s 恢复到 %s\n", backupName, targetPath)

	return nil
}

func resolveBackupPath(backupDir, backupName string) (string, error) {
	name := strings.TrimSpace(backupName)
	if name == "" {
		return "", fmt.Errorf("备份文件名不能为空")
	}
	if filepath.Base(name) != name {
		return "", fmt.Errorf("备份文件名不合法: %s", backupName)
	}

	baseDir, err := filepath.Abs(backupDir)
	if err != nil {
		return "", fmt.Errorf("无法解析备份目录: %w", err)
	}
	path := filepath.Join(baseDir, name)
	resolvedPath, err := filepath.Abs(path)
	if err != nil {
		return "", fmt.Errorf("无法解析备份路径: %w", err)
	}
	if resolvedPath != path {
		return "", fmt.Errorf("备份文件名不合法: %s", backupName)
	}
	return resolvedPath, nil
}

func validateRestoreData(data []byte, targetKind, sourcePath string) error {
	if err := config.ValidateYAMLBytes(data, sourcePath); err != nil {
		return fmt.Errorf("备份文件校验失败: %w", err)
	}
	switch targetKind {
	case "clashctl":
		cfg := core.DefaultAppConfig()
		if err := yaml.Unmarshal(data, cfg); err != nil {
			return fmt.Errorf("备份文件校验失败: 无法解析 clashctl 配置: %w", err)
		}
		if err := app.ValidateManagedPaths(cfg); err != nil {
			return fmt.Errorf("备份文件校验失败: clashctl 配置路径不安全: %w", err)
		}
	default:
		if err := config.ValidateProxyCount(data); err != nil {
			return fmt.Errorf("备份文件校验失败: %w", err)
		}
	}
	return nil
}

func copyBackupFile(sourcePath, backupPath string) error {
	data, err := config.ReadConfigWithLimit(sourcePath)
	if err != nil {
		return err
	}
	if err := system.ValidateOutputPath(backupPath); err != nil {
		return err
	}
	return system.WriteFileAtomic(backupPath, data, 0600)
}

func formatSize(bytes int64) string {
	const unit = 1024
	if bytes < unit {
		return fmt.Sprintf("%d B", bytes)
	}
	div, exp := int64(unit), 0
	for n := bytes / unit; n >= unit; n /= unit {
		div *= unit
		exp++
	}
	return fmt.Sprintf("%.1f %cB", float64(bytes)/float64(div), "KMGTPE"[exp])
}
