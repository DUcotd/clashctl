package cmd

import (
	"encoding/json"
	"fmt"
	"io"
	"os"

	"clashctl/internal/mihomo"
)

// errorReporter is implemented by all JSON report structs that carry an Error field.
type errorReporter interface {
	SetError(msg string)
}

// finishReport writes a JSON report when jsonFlag is true and returns err.
func finishReport[T errorReporter](report T, err error, jsonFlag bool) error {
	if err != nil {
		report.SetError(err.Error())
	}
	if jsonFlag {
		if writeErr := writeJSON(report); writeErr != nil {
			return writeErr
		}
	}
	return err
}

type installJSONReport struct {
	Path       string `json:"path,omitempty"`
	Version    string `json:"version,omitempty"`
	ReleaseTag string `json:"release_tag,omitempty"`
	Installed  bool   `json:"installed"`
}

type geoDataFileJSONReport struct {
	Name       string `json:"name"`
	Downloaded bool   `json:"downloaded"`
	Skipped    bool   `json:"skipped"`
	Required   bool   `json:"required"`
	Source     string `json:"source,omitempty"`
	Error      string `json:"error,omitempty"`
}

type geoDataJSONReport struct {
	AlreadyReady bool                    `json:"already_ready"`
	Downloaded   int                     `json:"downloaded"`
	Files        []geoDataFileJSONReport `json:"files,omitempty"`
}

type proxyInventoryJSONReport struct {
	Loaded         int      `json:"loaded"`
	Current        string   `json:"current,omitempty"`
	Candidates     []string `json:"candidates,omitempty"`
	OnlyCompatible bool     `json:"only_compatible"`
}

type runtimeStartJSONReport struct {
	Binary            *installJSONReport        `json:"binary,omitempty"`
	GeoData           *geoDataJSONReport        `json:"geodata,omitempty"`
	GeoDataError      string                    `json:"geodata_error,omitempty"`
	StartedBy         string                    `json:"started_by,omitempty"`
	ServiceStopped    bool                      `json:"service_stopped"`
	ProcessStopped    bool                      `json:"process_stopped"`
	ControllerReady   bool                      `json:"controller_ready"`
	ControllerVersion string                    `json:"controller_version,omitempty"`
	Inventory         *proxyInventoryJSONReport `json:"inventory,omitempty"`
	InventoryError    string                    `json:"inventory_error,omitempty"`
	Warnings          []string                  `json:"warnings,omitempty"`
}

func writeJSON(v any) error {
	return writeJSONTo(os.Stdout, v)
}

func writeJSONTo(w io.Writer, v any) error {
	encoder := json.NewEncoder(w)
	encoder.SetIndent("", "  ")
	if err := encoder.Encode(v); err != nil {
		return fmt.Errorf("写入 JSON 输出失败: %w", err)
	}
	return nil
}

func buildRuntimeStartJSONReport(result *mihomo.StartResult) *runtimeStartJSONReport {
	if result == nil {
		return nil
	}
	report := &runtimeStartJSONReport{
		GeoDataError:      result.GeoDataError,
		StartedBy:         result.StartedBy,
		ServiceStopped:    result.ServiceStopped,
		ProcessStopped:    result.ProcessStopped,
		ControllerReady:   result.ControllerReady,
		ControllerVersion: result.ControllerVersion,
		InventoryError:    result.InventoryError,
		Warnings:          append([]string(nil), result.Warnings...),
	}
	if result.Binary != nil {
		report.Binary = &installJSONReport{
			Path:       result.Binary.Path,
			Version:    result.Binary.Version,
			ReleaseTag: result.Binary.ReleaseTag,
			Installed:  result.Binary.Installed,
		}
	}
	if result.GeoData != nil {
		report.GeoData = &geoDataJSONReport{
			AlreadyReady: result.GeoData.AlreadyReady,
			Downloaded:   result.GeoData.Downloaded,
			Files:        make([]geoDataFileJSONReport, 0, len(result.GeoData.Files)),
		}
		for _, file := range result.GeoData.Files {
			report.GeoData.Files = append(report.GeoData.Files, geoDataFileJSONReport{
				Name:       file.Name,
				Downloaded: file.Downloaded,
				Skipped:    file.Skipped,
				Required:   file.Required,
				Source:     file.Source,
				Error:      file.Error,
			})
		}
	}
	if result.Inventory != nil {
		report.Inventory = &proxyInventoryJSONReport{
			Loaded:         result.Inventory.Loaded,
			Current:        result.Inventory.Current,
			Candidates:     append([]string(nil), result.Inventory.Candidates...),
			OnlyCompatible: result.Inventory.OnlyCompatible,
		}
	}
	return report
}
