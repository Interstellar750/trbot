package task

import (
	"context"
	"fmt"
	"time"
	"trbot/utils"

	"github.com/reugn/go-quartz/quartz"
	"github.com/rs/zerolog"
)

type Task struct {
	Name    string
	Group   string
	Job     quartz.Job
	Trigger quartz.Trigger
}

// Scheduler is an exported `quartz.Scheduler` struct, for use with other unencapsulated methods.
var Scheduler quartz.Scheduler

// InitTaskHandler retrieves the zerolog from ctx as the logger and starts the scheduler.
func InitTaskHandler(ctx context.Context) error {
	logger := zerolog.Ctx(ctx)
	var err error

	// slogLogger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelDebug}))
	Scheduler, err = quartz.NewStdScheduler(
		// quartz.WithLogger(quartz_logger.NewSlogLogger(ctx, slogLogger)),
		quartz.WithLogger(NewZerologWappred(*logger)),
	)
	if err != nil {
		logger.Error().
			Err(err).
			Msg("failed to create scheduler")
		return err
	}

	// start the scheduler
	Scheduler.Start(ctx)

	logger.Info().Msg("Task handler initialized")

	return nil
}

// ScheduleTask schedules a task into `task.Scheduler`.
func ScheduleTask(ctx context.Context, task Task) error {
	if task.Group == "" {
		task.Group = "default"
	}

	logger := zerolog.Ctx(ctx).With().
		Str(utils.GetCurrentFuncName()).
		Str("name", task.Name).
		Str("group", task.Group).
		Logger()

	if task.Job == nil {
		logger.Error().Msg("This task has no job")
		return fmt.Errorf("this task has no job")
	}

	var pauseJob bool
	if task.Trigger == nil {
		task.Trigger = quartz.NewSimpleTrigger(348 * time.Minute)
		pauseJob = true
		if task.Group != "trbot" {
			logger.Warn().Msg("A task has no trigger, it will add a trigger with a 348-minute interval and pause immediately")
		}
	}

	err := Scheduler.ScheduleJob(
		quartz.NewJobDetail(
			task.Job,
			quartz.NewJobKeyWithGroup(task.Name, task.Group),
		),
		task.Trigger,
	)
	if err != nil {
		logger.Error().
			Err(err).
			Msg("Failed to schedule task")
		return err
	}

	if pauseJob {
		err = Scheduler.PauseJob(quartz.NewJobKeyWithGroup(task.Name, task.Group))
		if err != nil {
			logger.Error().
				Err(err).
				Msg("Failed to pause a task that did not have a trigger")
		}
	}

	logger.Info().
		Bool("noTrigger", pauseJob).
		Msg("Task scheduled successfully")

	return nil
}

// UpdateTask updates a specified task by name and group.
func UpdateTask(ctx context.Context, name, group string, job quartz.Job, trigger quartz.Trigger) error {
	if group == "" {
		group = "default"
	}

	logger := zerolog.Ctx(ctx).With().
		Str(utils.GetCurrentFuncName()).
		Str("name", name).
		Str("group", group).
		Logger()

	if job == nil {
		logger.Error().Msg("No job in the newTask")
		return fmt.Errorf("no job in the newTask")
	}

	var pauseJob bool
	if trigger == nil {
		trigger = quartz.NewSimpleTrigger(348 * time.Minute)
		pauseJob = true
		logger.Warn().Msg("The newTask has no triggers, it will add a trigger with a 348-minute interval and pause immediately")
	}

	err := Scheduler.DeleteJob(quartz.NewJobKeyWithGroup(name, group))
	if err != nil {
		logger.Error().
			Err(err).
			Msg("Failed to delete old task")
		return fmt.Errorf("failed to delete old task: %w", err)
	}

	err = Scheduler.ScheduleJob(
		quartz.NewJobDetail(
			job,
			quartz.NewJobKeyWithGroup(name, group),
		),
		trigger,
	)
	if err != nil {
		logger.Error().
			Err(err).
			Msg("Failed to schedule newTask")
		return err
	}

	if pauseJob {
		err = Scheduler.PauseJob(quartz.NewJobKeyWithGroup(name, group))
		if err != nil {
			logger.Error().
				Err(err).
				Msg("Failed to pause a task that did not have a trigger")
		}
	}

	logger.Info().
		Bool("noTrigger", pauseJob).
		Msg("Task update successfully")

	return nil
}

// RunJobByName executes the job in `default` group.
func RunJobByName(ctx context.Context, name string) error {
	logger := zerolog.Ctx(ctx).
		With().
		Str(utils.GetCurrentFuncName()).
		Str("name", name).
		Logger()

	job, err := Scheduler.GetScheduledJob(quartz.NewJobKey(name))
	if err != nil {
		logger.Error().Err(err).Msg("Job not found")
		return err
	}

	err = job.JobDetail().Job().Execute(ctx)
	if err != nil {
		logger.Error().Err(err).Msg("Job execution failed")
		return err
	}

	return nil
}

// RunJobByNameAndGroup executes the job in specified group.
func RunJobByNameAndGroup(ctx context.Context, name, group string) error {
	logger := zerolog.Ctx(ctx).
		With().
		Str(utils.GetCurrentFuncName()).
		Str("name", name).
		Str("group", group).
		Logger()

	job, err := Scheduler.GetScheduledJob(quartz.NewJobKeyWithGroup(name, group))
	if err != nil {
		logger.Error().Err(err).Msg("Job not found")
		return err
	}

	err = job.JobDetail().Job().Execute(ctx)
	if err != nil {
		logger.Error().Err(err).Msg("Job execution failed")
		return err
	}

	return nil
}
