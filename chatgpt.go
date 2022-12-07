package chatgpt

import (
	"errors"

	"github.com/Clinet/clinet_convos"
	"github.com/Clinet/clinet_features"
	"github.com/Clinet/clinet_storage"
	"github.com/m1guelpf/chatgpt-telegram/src/chatgpt"
)

var Feature = features.Feature{
	Name: "chatgpt",
	Desc: "ChatGPT is available as a conversation service. You can @Clinet with a question, and ChatGPT may answer it!",
	ServiceConvo: &ClientChatGPT{},
}

type ClientChatGPT struct {
	sync.Mutex
	locked bool

	Client *chatgpt.ChatGPT
}

func (gpt *ClientChatGPT) unlock() {
	gpt.Unlock()
	gpt.locked = false
}

func (gpt *ClientChatGPT) Login() error {
	cfg := &storage.Storage{}
	if err := cfg.LoadFrom("chatgpt"); err != nil {
		return err
	}

	sessionToken := ""
	rawSessionToken, err := cfg.ConfigGet("cfg", "sessionToken")
	if err != nil {
		cfg.ConfigSet("cfg", "sessionToken", sessionToken)
	} else {
		sessionToken = rawSessionToken.(string)
	}

	gpt.Client = &chatgpt.ChatGPT{
		SessionToken: sessionToken,
	}
	return gpt.Client.EnsureAuth()
}

func (gpt *ClientChatGPT) Query(query *convos.ConversationQuery, lastState *convos.ConversationState) (*convos.ConversationResponse, error) {
	if gpt.locked {
		return nil, errors.New("too busy right now, sorry")
	}
	gpt.Lock()
	gpt.locked = true
	defer gpt.unlock()
	resp := &convos.ConversationResponse{}
	if lastState != nil {
		resp.ChatGPT = lastState.Response.ChatGPT
	}

	chanResult, err := gpt.Client.SendMessage(query.Text, resp.ChatGPT.ConversationId, resp.ChatGPT.MessageId)
	if err != nil {
		return nil, err
	}

	chatResp := <- chanResult

	resp.TextSimple = chatResp.Message
	resp.ChatGPT = chatResp

	return resp, nil
}