package task

type Status struct {
	Total        int64
	Current      int64
	StageTotal   int64
	StageCurrent int64
	Flags        uint64
}
