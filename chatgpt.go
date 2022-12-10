package chatgpt

import (
	"fmt"
	"sync"
	"time"

	"github.com/Clinet/clinet_convos"
	"github.com/Clinet/clinet_features"
	"github.com/Clinet/clinet_storage"
	"github.com/JoshuaDoes/logger"
	"github.com/m1guelpf/chatgpt-telegram/src/chatgpt"
	"github.com/m1guelpf/chatgpt-telegram/src/expirymap"
)

var Feature = features.Feature{
	Name: "chatgpt",
	Desc: "ChatGPT is available as a conversation service. You can @Clinet with a question, and ChatGPT may answer it!",
	ServiceConvo: &ClientChatGPT{},
}

var Log *logger.Logger
func init() {
	Log = logger.NewLogger("chatgpt", 2)
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
		return err
	} else {
		sessionToken = rawSessionToken.(string)
	}

	gpt.Client = &chatgpt.ChatGPT{
		AccessTokenMap: expirymap.New(),
		SessionToken: sessionToken,
	}
	return gpt.Client.EnsureAuth()
	//return nil
}

func (gpt *ClientChatGPT) Query(query *convos.ConversationQuery, lastState *convos.ConversationState) (*convos.ConversationResponse, error) {
	if gpt.locked {
		return nil, fmt.Errorf("too busy right now, sorry")
	}
	/*gpt.Lock()
	gpt.locked = true
	defer gpt.unlock()*/

	resp := &convos.ConversationResponse{}
	conversationID := ""
	messageID := ""
	if lastState != nil {
		resp = lastState.Response
		resp.TextSimple = ""
		conversationID = resp.ChatGPT.ConversationId
		messageID = resp.ChatGPT.MessageId
	}

	chanResult := make(chan chatgpt.ChatResponse)
	chanResultGot := false
	retryCount := 2
	for i := 0; i < retryCount; i++ {
		chanResultGet, err := gpt.Client.SendMessage(query.Text, conversationID, messageID)
		if err != nil {
			Log.Error(fmt.Sprintf("Sleeping for 10 seconds after attempt %d: %v", i+1, err))
			if i < (retryCount-1) {
				time.Sleep(time.Second * 10)
			}
			continue //Try again until we get literally any other error
		}
		chanResult = chanResultGet
		chanResultGot = true
		break
	}
	if !chanResultGot {
		Log.Error("Failed to get chanResult")
		return nil, fmt.Errorf("chatgpt: Failed to get chanResult")
	}

	Log.Trace("waiting")
	recv := false
	for chatResp := range chanResult {
		if !recv {
			recv = true
			Log.Trace("receiving")
		}
		resp.ChatGPT = chatResp
	}
	if !recv || resp.ChatGPT.Message == "" {
		return nil, fmt.Errorf("chatgpt: empty response")
	}

	resp.TextSimple = resp.ChatGPT.Message
	return resp, nil
}
