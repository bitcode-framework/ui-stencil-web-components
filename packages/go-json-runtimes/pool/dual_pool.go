package pool

import "context"

type DualPoolConfig struct {
	Worker     Config
	Background Config
}

type DualPool struct {
	worker     *Pool
	background *Pool
}

func NewDualPool(config DualPoolConfig, factory ProcessFactory) *DualPool {
	return &DualPool{
		worker:     NewPool(config.Worker, factory),
		background: NewPool(config.Background, factory),
	}
}

func (dp *DualPool) Execute(ctx context.Context, poolName string, request []byte) ([]byte, error) {
	switch poolName {
	case "background":
		return dp.background.Execute(ctx, request)
	default:
		return dp.worker.Execute(ctx, request)
	}
}

func (dp *DualPool) Worker() *Pool     { return dp.worker }
func (dp *DualPool) Background() *Pool { return dp.background }

func (dp *DualPool) Close() error {
	err1 := dp.worker.Close()
	err2 := dp.background.Close()
	if err1 != nil {
		return err1
	}
	return err2
}
