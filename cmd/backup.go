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
)

var backupCmd = &cobra.Command{
	Use:   "backup",
	Short: "备份当前配置",
	Long:  `备份 Mihomo 配置文件到 ~/.clashctl/backup/ 目录。`,
	RunE:  runBackup,
}

var restoreCmd = &cobra.Command{
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
	backupCmd.AddCommand(backupListCmd)
	rootCmd.AddCommand(backupCmd)
	rootCmd.AddCommand(restoreCmd)
}

// BackupDir returns the backup directory path.
func BackupDir() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("无法获取用户主目录: %w", err)
	}
	return filepath.Join(home, ".config", "clashctl", "backup"), nil
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

	// Backup mihomo config
	configPath := filepath.Join(cfg.ConfigDir, "config.yaml")
	if _, err := os.Stat(configPath); err == nil {
		timestamp := time.Now().Format("20060102-150405")
		backupPath := filepath.Join(backupDir, fmt.Sprintf("config-%s.yaml", timestamp))

		data, err := os.ReadFile(configPath)
		if err != nil {
			return fmt.Errorf("读取配置失败: %w", err)
		}

		if err := os.WriteFile(backupPath, data, 0600); err != nil {
			return fmt.Errorf("写入备份失败: %w", err)
		}

		fmt.Printf("✅ 配置已备份到: %s\n", backupPath)
	} else {
		fmt.Println("⚠️  未找到 Mihomo 配置文件")
	}

	// Backup clashctl config
	clashctlConfigPath, err := app.ConfigPath()
	if err == nil {
		if _, err := os.Stat(clashctlConfigPath); err == nil {
			timestamp := time.Now().Format("20060102-150405")
			backupPath := filepath.Join(backupDir, fmt.Sprintf("clashctl-%s.yaml", timestamp))

			data, err := os.ReadFile(clashctlConfigPath)
			if err == nil {
				if err := os.WriteFile(backupPath, data, 0600); err == nil {
					fmt.Printf("✅ clashctl 配置已备份到: %s\n", backupPath)
				}
			}
		}
	}

	return nil
}

func runBackupList(cmd *cobra.Command, args []string) error {
	backupDir, err := BackupDir()
	if err != nil {
		return err
	}

	if _, err := os.Stat(backupDir); os.IsNotExist(err) {
		fmt.Println("没有备份文件")
		return nil
	}

	entries, err := os.ReadDir(backupDir)
	if err != nil {
		return fmt.Errorf("读取备份目录失败: %w", err)
	}

	if len(entries) == 0 {
		fmt.Println("没有备份文件")
		return nil
	}

	fmt.Println("📦 可用备份:")
	fmt.Println()

	// Sort by modification time (newest first)
	type backupEntry struct {
		Name    string
		ModTime time.Time
		Size    int64
	}

	var backups []backupEntry
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		info, err := entry.Info()
		if err != nil {
			continue
		}
		backups = append(backups, backupEntry{
			Name:    entry.Name(),
			ModTime: info.ModTime(),
			Size:    info.Size(),
		})
	}

	sort.Slice(backups, func(i, j int) bool {
		return backups[i].ModTime.After(backups[j].ModTime)
	})

	for _, b := range backups {
		fmt.Printf("  %-40s  %s  %s\n", b.Name, b.ModTime.Format("2006-01-02 15:04:05"), formatSize(b.Size))
	}

	fmt.Println()
	fmt.Println("使用 'clashctl restore <文件名>' 恢复配置")

	return nil
}

func runRestore(cmd *cobra.Command, args []string) error {
	backupDir, err := BackupDir()
	if err != nil {
		return err
	}

	// If no args, list available backups
	if len(args) == 0 {
		return runBackupList(cmd, args)
	}

	backupName := args[0]
	backupPath := filepath.Join(backupDir, backupName)

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
	data, err := os.ReadFile(backupPath)
	if err != nil {
		return fmt.Errorf("读取备份失败: %w", err)
	}

	// Backup current config before restoring
	if _, err := os.Stat(targetPath); err == nil {
		if _, err := config.BackupFile(targetPath); err != nil {
			fmt.Printf("⚠️  备份当前配置失败: %v\n", err)
		}
	}

	// Restore
	if err := os.WriteFile(targetPath, data, 0600); err != nil {
		return fmt.Errorf("恢复配置失败: %w", err)
	}

	fmt.Printf("✅ 配置已从 %s 恢复到 %s\n", backupName, targetPath)

	return nil
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
