package dreamactivity

import (
	"arch-agent/internal/app/activity"
	"arch-agent/internal/app/dreaming"
	"context"
	"errors"
	"time"
)

type DreamActivityUseCase struct {
	dreamingService *dreaming.Service
	activityService *activity.Service
}

func NewDreamActivityUseCase(
	dreamingService *dreaming.Service,
	activityService *activity.Service,
) *DreamActivityUseCase {
	return &DreamActivityUseCase{
		dreamingService: dreamingService,
		activityService: activityService,
	}
}

func (uc *DreamActivityUseCase) Execute(ctx context.Context) error {
	data, err := uc.undreamedActivities()
	if err != nil {
		return err
	}

	for _, d := range data {
		if err := uc.dreamingService.Dream(ctx, d); err != nil {
			return err
		}
	}

	return nil
}

func (uc *DreamActivityUseCase) undreamedActivities() ([]string, error) {
	lastDate, err := uc.dreamingService.LastDreaming()
	if err != nil {

		return nil, err
	}
	return uc.getActivities(lastDate)
}

func (uc *DreamActivityUseCase) getActivities(from time.Time) ([]string, error) {
	result := []string{}
	for _, date := range dateRange(from) {
		data, err := uc.getActivity(date)
		if err != nil {
			return nil, err
		}
		result = append(result, data)
	}

	return result, nil
}

func (uc *DreamActivityUseCase) getActivity(on time.Time) (string, error) {
	data, err := uc.activityService.GetActivity(on)
	if err != nil {
		if errors.Is(err, activity.ErrNotFound) {
			return "", nil
		}
		return "", err
	}
	return on.Format("2006-01-02") + "\n" + data, nil
}

func dateRange(from time.Time) []time.Time {
	var dates []time.Time
	for d := from; !d.After(time.Now()); d = d.AddDate(0, 0, 1) {
		dates = append(dates, d)
	}
	return dates
}
