package handlers

import (
	"fmt"
	"log"
	"math/rand"
	"strings"
	"time"

	"github.com/eatmoreapple/openwechat"
	"github.com/qingconglaixueit/wechatbot/config"
	"github.com/qingconglaixueit/wechatbot/gpt"
	"github.com/qingconglaixueit/wechatbot/pkg/logger"
	"github.com/qingconglaixueit/wechatbot/service"
)

var _ MessageHandlerInterface = (*UserMessageHandler)(nil)

// UserMessageHandler 私聊消息处理
type UserMessageHandler struct {
	// 接收到消息
	msg *openwechat.Message
	// 发送的用户
	sender *openwechat.User
	// 实现的用户业务
	service service.UserServiceInterface
}

func UserMessageContextHandler() func(ctx *openwechat.MessageContext) {
	return func(ctx *openwechat.MessageContext) {
		msg := ctx.Message
		handler, err := NewUserMessageHandler(msg)
		if err != nil {
			logger.Warning(fmt.Sprintf("init user message handler error: %s", err))
		}

		// 处理用户消息
		err = handler.handle()
		if err != nil {
			logger.Warning(fmt.Sprintf("handle user message error: %s", err))
		}
	}
}

// NewUserMessageHandler 创建私聊处理器
func NewUserMessageHandler(message *openwechat.Message) (MessageHandlerInterface, error) {
	sender, err := message.Sender()
	if err != nil {
		return nil, err
	}
	userService := service.NewUserService(c, sender)
	handler := &UserMessageHandler{
		msg:     message,
		sender:  sender,
		service: userService,
	}

	return handler, nil
}

// handle 处理消息
func (h *UserMessageHandler) handle() error {
	if h.msg.IsText() {
		return h.ReplyText()
	}
	return nil
}

// ReplyText 发送文本消息到群
func (h *UserMessageHandler) ReplyText() error {
	if time.Now().Unix()-h.msg.CreateTime > 60 {
		return nil
	}

	maxInt := rand.New(rand.NewSource(time.Now().UnixNano())).Intn(5)
	time.Sleep(time.Duration(maxInt+1) * time.Second)

	log.Printf("Received User[%v], Content[%v], CreateTime[%v]", h.sender.NickName, h.msg.Content,
		time.Unix(h.msg.CreateTime, 0).Format("2006/01/02 15:04:05"))

	var (
		reply string
		err   error
	)
	messages := h.getRequestMessages()

	reply, err = gpt.Completions(messages)
	if err != nil {
		text := err.Error()
		if strings.Contains(err.Error(), "context deadline exceeded") {
			text = deadlineExceededText
		}
		_, err = h.msg.ReplyText(text)
		if err != nil {
			return fmt.Errorf("reply user error: %v ", err)
		}
		return err
	}

	h.service.SetUserSessionContext(h.msg.Content, reply)
	_, err = h.msg.ReplyText(buildUserReply(reply))
	if err != nil {
		return fmt.Errorf("reply user error: %v ", err)
	}

	// 3.返回错误
	return err
}

// getRequestMessages 获取请求接口的消息
func (h *UserMessageHandler) getRequestMessages() []gpt.Message {
	requestText := strings.TrimSpace(h.msg.Content)
	requestText = strings.Trim(h.msg.Content, "\n")

	messages := h.service.GetUserSessionContext()
	messages = append(messages, gpt.Message{
		Role:   "user",
		Content: requestText,
	})

	return messages
}

// buildUserReply 构建用户回复
func buildUserReply(reply string) string {
	// 1.去除空格问号以及换行号，如果为空，返回一个默认值提醒用户
	textSplit := strings.Split(reply, "\n\n")
	if len(textSplit) > 1 {
		trimText := textSplit[0]
		reply = strings.Trim(reply, trimText)
	}
	reply = strings.TrimSpace(reply)
	if reply == "" {
		return deadlineExceededText
	}

	// 2.如果用户有配置前缀，加上前缀
	reply = config.LoadConfig().ReplyPrefix + "\n" + reply
	reply = strings.Trim(reply, "\n")

	// 3.返回拼接好的字符串
	return reply
}
