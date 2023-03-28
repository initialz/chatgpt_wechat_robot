package handlers

import (
	"fmt"
	"log"
	"math/rand"
	"strings"
	"time"

	"github.com/eatmoreapple/openwechat"
	"github.com/qingconglaixueit/wechatbot/gpt"
	"github.com/qingconglaixueit/wechatbot/pkg/logger"
	"github.com/qingconglaixueit/wechatbot/service"
)

var _ MessageHandlerInterface = (*GroupMessageHandler)(nil)

// GroupMessageHandler 群消息处理
type GroupMessageHandler struct {
	// 获取自己
	self *openwechat.Self
	// 群
	group *openwechat.Group
	// 接收到消息
	msg *openwechat.Message
	// 发送的用户
	sender *openwechat.User
	// 实现的用户业务
	service service.UserServiceInterface
}

func GroupMessageContextHandler() func(ctx *openwechat.MessageContext) {
	return func(ctx *openwechat.MessageContext) {
		msg := ctx.Message
		// 获取用户消息处理器
		handler, err := NewGroupMessageHandler(msg)
		if err != nil {
			logger.Warning(fmt.Sprintf("init group message handler error: %v", err))
			return
		}

		// 处理用户消息
		err = handler.handle()
		if err != nil {
			logger.Warning(fmt.Sprintf("handle group message error: %v", err))
		}
	}
}

// NewGroupMessageHandler 创建群消息处理器
func NewGroupMessageHandler(msg *openwechat.Message) (MessageHandlerInterface, error) {
	sender, err := msg.Sender()
	if err != nil {
		return nil, err
	}
	group := &openwechat.Group{User: sender}
	groupSender, err := msg.SenderInGroup()
	if err != nil {
		return nil, err
	}

	userService := service.NewUserService(c, groupSender)
	handler := &GroupMessageHandler{
		self:    sender.Self,
		msg:     msg,
		group:   group,
		sender:  groupSender,
		service: userService,
	}
	return handler, nil

}

// handle 处理消息
func (g *GroupMessageHandler) handle() error {
	if g.msg.IsText() {
		return g.ReplyText()
	}
	return nil
}

// ReplyText 发息送文本消到群
func (g *GroupMessageHandler) ReplyText() error {
	if time.Now().Unix()-g.msg.CreateTime > 60 {
		return nil
	}

	maxInt := rand.New(rand.NewSource(time.Now().UnixNano())).Intn(5)
	time.Sleep(time.Duration(maxInt+1) * time.Second)

	log.Printf("Received Group[%v], Content[%v], CreateTime[%v]", g.group.NickName, g.msg.Content,
		time.Unix(g.msg.CreateTime, 0).Format("2006/01/02 15:04:05"))

	var (
		err   error
		reply string
	)

	// 1.不是@的不处理
	if !g.msg.IsAt() {
		return nil
	}

	// 2.获取请求的文本，如果为空字符串不处理
	messages := g.getRequestMessages()

	// 3.请求GPT获取回复
	reply, err = gpt.Completions(messages)
	if err != nil {
		text := err.Error()
		if strings.Contains(err.Error(), "context deadline exceeded") {
			text = deadlineExceededText
		}
		_, err = g.msg.ReplyText(text)
		if err != nil {
			return fmt.Errorf("reply group error: %v", err)
		}
		return err
	}

	// 4.设置上下文，并响应信息给用户
	g.service.SetUserSessionContext(g.msg.Content, reply)
	_, err = g.msg.ReplyText(g.buildReplyText(reply))
	if err != nil {
		return fmt.Errorf("reply group error: %v ", err)
	}

	// 5.返回错误信息
	return err
}

// getRequestMessages 获取请求接口的消息
func (h *GroupMessageHandler) getRequestMessages() []gpt.Message {
	requestText := strings.TrimSpace(h.msg.Content)
	requestText = strings.Trim(h.msg.Content, "\n")

	messages := h.service.GetUserSessionContext()
	messages = append(messages, gpt.Message{
		Role:   "user",
		Content: requestText,
	})

	return messages
}

// buildReply 构建回复文本
func (g *GroupMessageHandler) buildReplyText(reply string) string {
	// 1.获取@我的用户
	atText := "@" + g.sender.NickName
	textSplit := strings.Split(reply, "\n\n")
	if len(textSplit) > 1 {
		trimText := textSplit[0]
		reply = strings.Trim(reply, trimText)
	}
	reply = strings.TrimSpace(reply)
	if reply == "" {
		return atText + " " + deadlineExceededText
	}

	// 2.拼接回复, @我的用户, 问题, 回复
	replaceText := "@" + g.self.NickName
	question := strings.TrimSpace(strings.ReplaceAll(g.msg.Content, replaceText, ""))
	hr := strings.Repeat("-", 36)
	reply = atText + "\n" + question + "\n" + hr + "\n" + reply
	reply = strings.Trim(reply, "\n")

	// 3.返回回复的内容
	return reply
}
