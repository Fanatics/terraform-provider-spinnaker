package client

import "github.com/mitchellh/mapstructure"

func parseDestroyServerGroupStage(stageMap map[string]interface{}) (Stage, error) {
	stage := newSerializableTargetServerGroupStage(DestroyServerGroupStageType)
	if err := mapstructure.Decode(stageMap, stage); err != nil {
		return nil, err
	}

	notifications, err := parseNotifications(stageMap["notifications"])
	if err != nil {
		return nil, err
	}
	return &TargetServerGroupStage{
		serializableTargetServerGroupStage: stage,
		Notifications:                      notifications,
	}, nil
}
