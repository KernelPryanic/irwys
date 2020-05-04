package irwys

import (
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"math/rand"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"time"

	tgbotapi "github.com/Syfaro/telegram-bot-api"
	"github.com/smallfish/simpleyaml"
	"github.com/syndtr/goleveldb/leveldb/opt"
)

var (
	Verbose *log.Logger
	Info    *log.Logger
	Warning *log.Logger
	Error   *log.Logger
)

var chats = NewSynMap()
var replies = NewSynMap()
var lock = sync.RWMutex{}

func Init(
	verboseHandle io.Writer,
	infoHandle io.Writer,
	warningHandle io.Writer,
	errorHandle io.Writer,
) {
	Verbose = log.New(verboseHandle,
		"VERBOSE: ",
		log.Ldate|log.Ltime|log.Lshortfile)

	Info = log.New(infoHandle,
		"INFO: ",
		log.Ldate|log.Ltime|log.Lshortfile)

	Warning = log.New(warningHandle,
		"WARNING: ",
		log.Ldate|log.Ltime|log.Lshortfile)

	Error = log.New(errorHandle,
		"ERROR: ",
		log.Ldate|log.Ltime|log.Lshortfile)
}

type bot struct {
	token string
	opts  *Options
}

func New(
	token string,
	opts *Options,
) bot {
	b := bot{token, opts}
	return b
}

func handleRememberErr(err error, update tgbotapi.Update) {
	if err != nil {
		Error.Printf("Couldn't remember message\n\tChatId: %d\n\tMessage ID: %d",
			update.Message.Chat.ID, update.Message.MessageID)
	}
}

func handleRecallErr(err error, update tgbotapi.Update) {
	if err != nil {
		Error.Printf("Couldn't recall messages\n\tChatId: %d", update.Message.Chat.ID)
	}
}

func welcome(update tgbotapi.Update, botAPI *tgbotapi.BotAPI) {
	reply := fmt.Sprintf(
		"*I Remember What You Said bot*\n\n" +

			"This bot prowls through the chat history and racalls some messages time to time.\n\n" +

			"*Commands you can use:*\n\n" +

			"/start - start the bot\n" +
			"/stop - stop the bot\n" +
			"/recall - recall random message\n" +
			"/ru | /en - change the language\n" +
			"/help - show this message",
	)

	msg := tgbotapi.NewMessage(update.Message.Chat.ID, reply)
	msg.ParseMode = "markdown"
	_, err := botAPI.Send(msg)
	if err != nil {
		Error.Printf("Can't send reply to %s\n\tError: %s",
			update.Message.From.UserName, err)
	}
}

func (b bot) remember(dbMessages DB, update tgbotapi.Update) {
	if update.Message.Chat.IsChannel() {
		Warning.Printf("Do not work with channels.")
		return
	}

	l := len(strings.Split(update.Message.Text, " "))
	if (l < int(b.opts.minWords) || l > int(b.opts.maxWords)) && update.Message.Photo == nil {
		return
	}

	chatIDStr := strconv.FormatInt(update.Message.Chat.ID, 10)
	m, err := dbMessages.Get(chatIDStr)
	handleRecallErr(err, update)
	if m == nil {
		m = []int{}
	}
	messages := m.([]int)
	if len(messages) >= int(b.opts.capacity) {
		messages = messages[len(messages)-int(b.opts.capacity)+1:]
	}
	err = dbMessages.Put(chatIDStr, append(messages, update.Message.MessageID))
	handleRememberErr(err, update)
}

func (b bot) recall(dbMessages DB, dbChats DB, update tgbotapi.Update, botAPI *tgbotapi.BotAPI) {
	if update.Message.Chat.IsChannel() {
		Warning.Printf("Can't send reply to channel %s", update.Message.From.UserName)
		return
	}

	var lang = "en"

	rand.Seed(time.Now().UTC().UnixNano())
	chatIDStr := strconv.FormatInt(update.Message.Chat.ID, 10)
	rawMessages, err := dbMessages.Get(chatIDStr)
	handleRecallErr(err, update)

	if rawMessages == nil {
		return
	}
	evalMessages := rawMessages.([]int)

	rawConf, err := dbChats.Get(strconv.FormatInt(update.Message.Chat.ID, 10))
	if rawConf != nil {
		evalConf := rawConf.(map[string]string)
		lang = evalConf["language"]
	}
	if err != nil {
		Error.Printf("Can't get chat reply language\n\tChatId: %d", update.Message.Chat.ID)
	}

	fwdMessageID := rand.Intn(len(evalMessages))
	fwdMsg := tgbotapi.NewForward(update.Message.Chat.ID,
		update.Message.Chat.ID, evalMessages[fwdMessageID])
	sent, err := botAPI.Send(fwdMsg)
	if err != nil {
		Error.Printf("Can't forward message\n\tChatId: %d\n\t%s", update.Message.Chat.ID, err)
	}

	possibleReplies := replies.Get(lang).(map[interface{}]interface{})
	msgType := "text"
	if sent.Photo != nil {
		msgType = "photo"
	}
	replyID := rand.Intn(len(possibleReplies[msgType].([]interface{})))
	msg := tgbotapi.NewMessage(update.Message.Chat.ID, possibleReplies[msgType].([]interface{})[replyID].(string))
	_, err = botAPI.Send(msg)
	if err != nil {
		Error.Printf("Can't send message\n\tChatId: %d\n\t%s", update.Message.Chat.ID, err)
	}

	Verbose.Printf("Recalled\n\tChatId: %d\n\tFwdMessageId: %d", update.Message.Chat.ID, evalMessages[fwdMessageID])
}

func (b bot) language(dbChats DB, update tgbotapi.Update) (err error) {
	chatIDStr := strconv.FormatInt(update.Message.Chat.ID, 10)
	lang := update.Message.Text
	lang = strings.Replace(lang, "/", "", -1)

	rawConf, err := dbChats.Get(chatIDStr)
	if err != nil {
		Error.Printf("Can't get chat information\n\tChatId: %d\n\t%s",
			update.Message.Chat.ID, err)
	}
	if rawConf == nil {
		rawConf = map[string]string{}
	}
	evalConf := rawConf.(map[string]string)

	evalConf["language"] = lang
	err = dbChats.Put(chatIDStr, evalConf)
	if err != nil {
		Error.Printf("Can't set language\n\tChatId: %d\n\tLanguage: %s\n\t%s",
			update.Message.Chat.ID, update.Message.Text, err)
	}

	return
}

func (b bot) watcher(dbMessages DB, dbChats DB, ch chan tgbotapi.Update, botAPI *tgbotapi.BotAPI) {
	defer close(ch)

	var lastUpdateDate time.Time
	var update tgbotapi.Update
	var ok = true
	rand.Seed(time.Now().UTC().UnixNano())

	for {
		select {
		case update, ok = <-ch:
			if ok == false {
				break
			}
			b.remember(dbMessages, update)
			lastUpdateDate = time.Unix(int64(update.Message.Date), 0)
		default:
			now := time.Now()
			if now.Hour() < int(b.opts.timeStart) || now.Hour() >= int(b.opts.timeEnd) ||
				update.Message == nil {
				continue
			}
			acceptableWindow := now.Add(time.Duration(-b.opts.timeout) * time.Minute)
			if !lastUpdateDate.After(acceptableWindow) {
				if rand.Float64() < 0.3 {
					b.recall(dbMessages, dbChats, update, botAPI)
				}
				lastUpdateDate = now
			}
		}
		time.Sleep(time.Duration(1) * time.Millisecond)
	}
}

func (b bot) start(dbChats DB, update tgbotapi.Update) {
	chatIDStr := strconv.FormatInt(update.Message.Chat.ID, 10)

	if exist, _ := dbChats.Exist(chatIDStr); exist {
		return
	}

	update.Message.Text = "en"
	if err := b.language(dbChats, update); err != nil {
		Error.Printf("Failed to start\n\tChatId: %d\n\tError: %s",
			update.Message.Chat.ID, err)
	} else {
		Info.Printf("Bot successfully started\n\tChatId: %d", update.Message.Chat.ID)
	}
}

func (b bot) stop(dbChats DB, update tgbotapi.Update) {
	chatIDStr := strconv.FormatInt(update.Message.Chat.ID, 10)

	if exist, _ := dbChats.Exist(chatIDStr); !exist {
		return
	}

	err := dbChats.Delete(chatIDStr)
	if err != nil {
		Error.Printf("Can't remove chat\n\tChatId: %d\n\tError: %s",
			update.Message.Chat.ID, err)
	}

	if chats.Exist(chatIDStr) {
		c := chats.Get(chatIDStr).(chan tgbotapi.Update)
		close(c)
		chats.Delete(chatIDStr)
	}

	if err != nil {
		Error.Printf("Failed to stop\n\tChatId: %d\n\tError: %s",
			update.Message.Chat.ID, err)
	} else {
		Info.Printf("Bot successfully stopped\n\tChatId: %d", update.Message.Chat.ID)
	}
}

func (b bot) initBot(dbMessages DB, dbChats DB, botAPI *tgbotapi.BotAPI) {
	for _, lang := range []string{"en", "ru"} {
		rawData, err := ioutil.ReadFile(filepath.Join(b.opts.replyPath, fmt.Sprintf("%s.yml", lang)))
		if err != nil {
			Error.Printf("Can't open file with replies\n\tPath: %s\n\tError: %s", b.opts.replyPath, err)
		}

		ymlData, err := simpleyaml.NewYaml(rawData)
		if err != nil {
			Error.Printf("Can't convert replies to YAML\n\tPath: %s\n\tError: %s", b.opts.replyPath, err)
		}

		m, _ := ymlData.Map()
		replies.Put(lang, m)
	}

	for it := dbChats.Iterate(nil); it.Next(); {
		k, _ := it.Key(), it.Value()
		ch := make(chan tgbotapi.Update, 1)
		chats.Put(string(k), ch)
		go b.watcher(dbMessages, dbChats, ch, botAPI)
	}
}

func (b bot) Start() {
	if b.opts.verbose {
		Init(os.Stdout, os.Stdout, os.Stdout, os.Stderr)
	} else {
		Init(ioutil.Discard, os.Stdout, os.Stdout, os.Stderr)
	}

	o := &opt.Options{
		BlockCacheCapacity: 128 * opt.MiB,
		WriteBuffer:        16 * opt.MiB,
	}
	dbMessages := NewDB(b.opts.dbPath, "messages", o)
	defer dbMessages.Close()
	dbChats := NewDB(b.opts.dbPath, "chats", o)
	defer dbChats.Close()

	botAPI, err := tgbotapi.NewBotAPI(b.token)
	if err != nil {
		Error.Println("Can't authenticate with given token")
		panic(err)
	}

	Info.Printf("Authorized on account %s", botAPI.Self.UserName)

	b.initBot(dbMessages, dbChats, botAPI)

	u := tgbotapi.NewUpdate(0)
	u.Timeout = 60

	updates, err := botAPI.GetUpdatesChan(u)

	for update := range updates {
		if update.Message == nil {
			continue
		}

		chatIDStr := strconv.FormatInt(update.Message.Chat.ID, 10)

		Info.Printf("[%s] %s", update.Message.From.UserName, update.Message.Text)
		Verbose.Printf("ChatId: %d", update.Message.Chat.ID)

		switch update.Message.Command() {
		case "start":
			go welcome(update, botAPI)
			b.start(dbChats, update)
			ch := make(chan tgbotapi.Update, 1)
			chats.Put(chatIDStr, ch)
			go b.watcher(dbMessages, dbChats, ch, botAPI)
		case "stop":
			b.stop(dbChats, update)
		case "help":
			go welcome(update, botAPI)
		case "recall":
			go b.recall(dbMessages, dbChats, update, botAPI)
		case "ru", "en":
			go b.language(dbChats, update)
		}

		if chats.Exist(chatIDStr) {
			chats.Get(chatIDStr).(chan tgbotapi.Update) <- update
		}
	}
}
