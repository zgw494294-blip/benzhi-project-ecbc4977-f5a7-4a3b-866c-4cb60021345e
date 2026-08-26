package domain

type TaskStatus string

const (
	StatusDraft     TaskStatus = "draft"
	StatusObserving TaskStatus = "observing"
	StatusAssessed  TaskStatus = "assessed"
	StatusRepairing TaskStatus = "repairing"
	StatusRetesting TaskStatus = "retesting"
	StatusReviewing TaskStatus = "reviewing"
	StatusReleased  TaskStatus = "released"
)

func (s TaskStatus) Label() string {
	switch s {
	case StatusDraft:
		return "草稿"
	case StatusObserving:
		return "观测采集"
	case StatusAssessed:
		return "风险已评估"
	case StatusRepairing:
		return "维修执行"
	case StatusRetesting:
		return "维修复测"
	case StatusReviewing:
		return "安全复核"
	case StatusReleased:
		return "已放行"
	default:
		return string(s)
	}
}

func CanTransition(from, to TaskStatus) bool {
	allowed := map[TaskStatus][]TaskStatus{
		StatusDraft:     {StatusObserving},
		StatusObserving: {StatusAssessed},
		StatusAssessed:  {StatusRepairing},
		StatusRepairing: {StatusRetesting},
		StatusRetesting: {StatusReviewing},
		StatusReviewing: {StatusRetesting, StatusReleased},
	}
	for _, candidate := range allowed[from] {
		if candidate == to {
			return true
		}
	}
	return false
}
