package sql_exporter

import (
	"errors"
	"log/slog"

	cfg "github.com/burningalchemist/sql_exporter/config"
)

// Reload function is used to reload the exporter configuration without restarting the exporter
func Reload(e Exporter, configFile *string) error {
	slog.Warn("Reloading collectors has started...")
	slog.Warn("Connections will not be changed upon the restart of the exporter")
	configNext, err := cfg.Load(*configFile)
	if err != nil {
		slog.Error("Error reading config file", "error", err)
		return err
	}

	configCurrent := e.Config()

	// Clear current collectors and replace with new ones
	if len(configCurrent.Collectors) > 0 {
		configCurrent.Collectors = configCurrent.Collectors[:0]
	}
	configCurrent.Collectors = configNext.Collectors
	slog.Debug("Total collector size change", "from", len(configCurrent.Collectors), "to", len(configNext.Collectors))

	// Reload targets
	switch {
	case configCurrent.Target != nil && configNext.Target != nil:
		if err = reloadTarget(e, configNext, configCurrent); err != nil {
			return err
		}
	case len(configCurrent.Jobs) > 0 && len(configNext.Jobs) > 0:
		if err = reloadJobs(e, configNext, configCurrent); err != nil {
			return err
		}
	case configCurrent.Target != nil && len(configNext.Jobs) > 0:
	case len(configCurrent.Jobs) > 0 && configNext.Target != nil:
		return errors.New("changing scrape mode is not allowed. Please restart the exporter")
	default:
		slog.Warn("No target or jobs have been found - nothing to reload")
	}
	return nil
}

func reloadTarget(e Exporter, nc, cc *cfg.Config) error {
	slog.Warn("Recreating target...")

	// We want to preserve DSN from the previous config revision to avoid any connection changes
	nc.Target.DSN = cc.Target.DSN
	// Apply the new target configuration
	cc.Target = nc.Target
	// Recreate the target object
	target, err := NewTarget("", cc.Target.Name, "", string(cc.Target.DSN),
		cc.Target.Collectors(), nil, cc.Globals, cc.Target.EnablePing)
	if err != nil {
		slog.Error("Error recreating a target", "error", err)
		return err
	}

	// Populate the target list
	e.UpdateTarget([]Target{target})
	slog.Warn("Collectors have been successfully updated for the target")
	return nil
}

func reloadJobs(e Exporter, nc, cc *cfg.Config) error {
	slog.Warn("Recreating jobs...")
	// We want to preserve `static_configs`` from the previous config revision to avoid any connection changes
	for _, currentJob := range cc.Jobs {
		for _, newJob := range nc.Jobs {
			if newJob.Name == currentJob.Name {
				newJob.StaticConfigs = currentJob.StaticConfigs
			}
		}
	}
	cc.Jobs = nc.Jobs
	var updateErr error
	targets := make([]Target, 0, len(cc.Jobs))

	for _, jobConfigItem := range cc.Jobs {
		job, err := NewJob(jobConfigItem, cc.Globals)
		if err != nil {
			updateErr = err
			break
		}
		targets = append(targets, job.Targets()...)
		slog.Debug("Recreated Job", "name", jobConfigItem.Name)
	}

	if updateErr != nil {
		slog.Error("Error recreating jobs", "error", updateErr)
		return updateErr
	}

	e.UpdateTarget(targets)
	slog.Warn("Collectors have been successfully updated for the jobs")
	return nil
}
