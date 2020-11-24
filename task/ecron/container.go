package ecron

import (
	"github.com/gotomicro/ego/client/eetcd"
	"github.com/gotomicro/ego/core/conf"
	"github.com/gotomicro/ego/core/elog"
	"github.com/robfig/cron/v3"
)

type Option func(c *Container)

type Container struct {
	config *Config
	name   string
	logger *elog.Component
}

func DefaultContainer() *Container {
	return &Container{
		logger: elog.EgoLogger.With(elog.FieldMod("task.ecron")),
	}
}

func Load(key string) *Container {
	c := DefaultContainer()
	var config = DefaultConfig()
	if err := conf.UnmarshalKey(key, &config); err != nil {
		c.logger.Panic("parse config error", elog.FieldErr(err), elog.FieldKey(key))
		return c
	}
	c.config = config
	c.name = key

	return c
}

func WithClientEtcd(etcdClient *eetcd.Component) Option {
	return func(c *Container) {
		c.config.etcdClient = etcdClient
	}
}

// WithChain ...
func WithChain(wrappers ...JobWrapper) Option {
	return func(c *Container) {
		if c.config.wrappers == nil {
			c.config.wrappers = []JobWrapper{}
		}
		c.config.wrappers = append(c.config.wrappers, wrappers...)
	}
}

// Build ...
func (c *Container) Build(options ...Option) *Component {
	for _, option := range options {
		option(c)
	}

	if c.config.WithSeconds {
		c.config.parser = cron.NewParser(cron.Second | cron.Minute | cron.Hour | cron.Dom | cron.Month | cron.Dow | cron.Descriptor)
	}

	if c.config.ConcurrentDelay > 0 { // 延迟
		c.config.wrappers = append(c.config.wrappers, delayIfStillRunning(c.logger))
	} else if c.config.ConcurrentDelay < 0 { // 跳过
		c.config.wrappers = append(c.config.wrappers, skipIfStillRunning(c.logger))
	} else {
		// 默认不延迟也不跳过
	}

	if c.config.DistributedTask && c.config.etcdClient == nil {
		c.logger.Panic("client etcd nil", elog.FieldKey("use WithClientEtcd method"))
	}

	return newComponent(c.name, c.config, c.logger)
}
