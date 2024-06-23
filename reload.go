package sql_exporter

import (
	"errors"

	cfg "github.com/burningalchemist/sql_exporter/config"
	"k8s.io/klog/v2"
)

func Reload(e Exporter, configFile *string) error {
	klog.Warning("Reloading collectors has started...")
	klog.Warning("Connections will not be changed upon the restart of the exporter")
	exporterNewConfig, err := cfg.Load(*configFile)
	if err != nil {
		klog.Errorf("Error reading config file - %v", err)
		return err
	}

	currentConfig := e.Config()

	// Clear current collectors and replace with new ones
	if len(currentConfig.Collectors) > 0 {
		currentConfig.Collectors = currentConfig.Collectors[:0]
	}
	currentConfig.Collectors = exporterNewConfig.Collectors
	klog.Infof("Total collector size change: %v -> %v", len(currentConfig.Collectors),
		len(exporterNewConfig.Collectors))

	// Reload targets
	switch {
	case currentConfig.Target != nil && exporterNewConfig.Target != nil:
		if err = reloadTarget(e, exporterNewConfig, currentConfig); err != nil {
			return err
		}
	case len(currentConfig.Jobs) > 0 && len(exporterNewConfig.Jobs) > 0:
		if err = reloadJobs(e, exporterNewConfig, currentConfig); err != nil {
			return err
		}
	case currentConfig.Target != nil && len(exporterNewConfig.Jobs) > 0:
	case len(currentConfig.Jobs) > 0 && exporterNewConfig.Target != nil:
		return errors.New("changing scrape mode is not allowed. Please restart the exporter")
	default:
		klog.Warning("No target or jobs have been found - nothing to reload")
	}
	return nil
}

func reloadTarget(e Exporter, nc, cc *cfg.Config) error {
	klog.Warning("Recreating targets collectors...")
	// FIXME: Should be t.Collectors() instead of config.Collectors
	target, err := NewTarget("", cc.Target.Name, "", string(cc.Target.DSN),
		nc.Target.Collectors(), nil, cc.Globals, cc.Target.EnablePing)
	if err != nil {
		klog.Errorf("Error recreating a target - %v", err)
		return err
	}

	e.UpdateTarget([]Target{target})
	klog.Warning("Collectors have been successfully reloaded for target")
	return nil
}

func reloadJobs(e Exporter, nc, cc *cfg.Config) error {
	klog.Warning("Recreating jobs...")

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
		klog.Infof("Recreated Job: %s", jobConfigItem.Name)
	}

	if updateErr != nil {
		klog.Errorf("Error recreating jobs - %v", updateErr)
		return updateErr
	}

	e.UpdateTarget(targets)
	klog.Warning("Query collectors have been successfully reloaded for jobs")
	return nil
}
