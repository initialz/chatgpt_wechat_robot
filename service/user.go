package service

import (
	"github.com/qingconglaixueit/wechatbot/config"
	"github.com/qingconglaixueit/wechatbot/gpt"
	"github.com/qingconglaixueit/wechatbot/pkg/logger"
	"github.com/eatmoreapple/openwechat"
	"github.com/patrickmn/go-cache"
	"time"
)

// UserServiceInterface 用户业务接口
type UserServiceInterface interface {
	GetUserSessionContext() []gpt.Message
	SetUserSessionContext(question, reply string)
	ClearUserSessionContext()
}

var _ UserServiceInterface = (*UserService)(nil)

// UserService 用戶业务
type UserService struct {
	// 缓存
	cache *cache.Cache
	// 用户
	user *openwechat.User
}

// NewUserService 创建新的业务层
func NewUserService(cache *cache.Cache, user *openwechat.User) UserServiceInterface {
	return &UserService{
		cache: cache,
		user:  user,
	}
}

// ClearUserSessionContext 清空GTP上下文，接收文本中包含`我要问下一个问题`，并且Unicode 字符数量不超过20就清空
func (s *UserService) ClearUserSessionContext() {
	s.cache.Delete(s.user.ID())
}

// GetUserSessionContext 获取用户会话上下文文本
func (s *UserService) GetUserSessionContext() []gpt.Message {
	history, ok := s.cache.Get(s.user.ID())
	if !ok {
		return []gpt.Message{}
	}

	return history.([]gpt.Message)	
}

// SetUserSessionContext 设置用户会话上下文文本，question用户提问内容，GTP回复内容
func (s *UserService) SetUserSessionContext(question, reply string) {
	history, _ := s.cache.Get(s.user.ID())
	logger.Info("history: ", history)
	// 如果history为空，初始化一个空的切片
	if history == nil {
		history = []gpt.Message{}
	}

	history = append(history.([]gpt.Message), gpt.Message{
		Role: "user",
		Content: question,
	})

	history = append(history.([]gpt.Message), gpt.Message{
		Role: "assistant",
		Content: reply,
	})

	s.cache.Set(s.user.ID(), history, time.Second*config.LoadConfig().SessionTimeout)
}
