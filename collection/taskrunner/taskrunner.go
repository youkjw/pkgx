package taskrunner

type TaskRunner struct {
	limit chan struct{}
}

func NewTaskRunner(parallel int) *TaskRunner {
	return &TaskRunner{limit: make(chan struct{}, parallel)}
}

func (t *TaskRunner) Schedule(task func()) {
	t.limit <- struct{}{}

	go func() {
		defer func() {
			<-t.limit
		}()

		task()
	}()
}
