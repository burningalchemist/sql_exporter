package sql_exporter

import (
	"errors"

	cfg "github.com/burningalchemist/sql_exporter/config"
	"k8s.io/klog/v2"
)

// Reload function is used to reload the exporter configuration without restarting the exporter
func Reload(e Exporter, configFile *string) error {
	klog.Warning("Reloading collectors has started...")
	klog.Warning("Connections will not be changed upon the restart of the exporter")
	configNext, err := cfg.Load(*configFile)
	if err != nil {
		klog.Errorf("Error reading config file - %v", err)
		return err
	}

	configCurrent := e.Config()

	// Clear current collectors and replace with new ones
	if len(configCurrent.Collectors) > 0 {
		configCurrent.Collectors = configCurrent.Collectors[:0]
	}
	configCurrent.Collectors = configNext.Collectors
	klog.Infof("Total collector size change: %v -> %v", len(configCurrent.Collectors),
		len(configNext.Collectors))

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
		klog.Warning("No target or jobs have been found - nothing to reload")
	}
	return nil
}

func reloadTarget(e Exporter, nc, cc *cfg.Config) error {
	klog.Warning("Recreating target...")

	// We want to preserve DSN from the previous config revision to avoid any connection changes
	nc.Target.DSN = cc.Target.DSN
	// Apply the new target configuration
	cc.Target = nc.Target
	// Recreate the target object
	target, err := NewTarget("", cc.Target.Name, "", string(cc.Target.DSN),
		cc.Target.Collectors(), nil, cc.Globals, cc.Target.EnablePing)
	if err != nil {
		klog.Errorf("Error recreating a target - %v", err)
		return err
	}

	// Populate the target list
	e.UpdateTarget([]Target{target})
	klog.Warning("Collectors have been successfully updated for the target")
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
	klog.Warning("Collectors have been successfully updated for the jobs")
	return nil
}
