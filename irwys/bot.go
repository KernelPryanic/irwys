package irwys

import (
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"math/rand"
	"os"
	"strconv"
	"strings"
	"sync"
	"time"

	tgbotapi "github.com/Syfaro/telegram-bot-api"
	"github.com/syndtr/goleveldb/leveldb/opt"
)

var (
	Verbose *log.Logger
	Info    *log.Logger
	Warning *log.Logger
	Error   *log.Logger
)

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

			"/recall - recall random message\n" +
			"/ru|/en - change the language\n" +
			"/help - show this message",
	)

	msg := tgbotapi.NewMessage(update.Message.Chat.ID, reply)
	msg.ParseMode = "markdown"
	_, err := botAPI.Send(msg)
	if err != nil {
		Error.Printf("Can't send reply to %s", update.Message.From.UserName)
	}
}

func (b bot) remember(db DB, update tgbotapi.Update) {
	if update.Message.Chat.IsChannel() {
		Warning.Printf("Do not work with channels.")
		return
	}

	l := len(strings.Split(update.Message.Text, " "))
	if (l < int(b.opts.minLength) || l > int(b.opts.maxLength)) && update.Message.Photo == nil {
		return
	}

	chatIDStr := strconv.FormatUint(uint64(update.Message.Chat.ID), 10)
	m, err := db.Get(chatIDStr)
	handleRecallErr(err, update)
	if m == nil {
		m = []int{}
	}
	messages := m.([]int)
	if len(messages) >= int(b.opts.capacity) {
		messages = messages[len(messages)-int(b.opts.capacity)+1:]
	}
	err = db.Put(chatIDStr, append(messages, update.Message.MessageID))
	handleRememberErr(err, update)
}

func (b bot) recall(db DB, update tgbotapi.Update, botAPI *tgbotapi.BotAPI) {
	if update.Message.Chat.IsChannel() {
		Warning.Printf("Can't send reply to channel %s", update.Message.From.UserName)
		return
	}

	rand.Seed(time.Now().UTC().UnixNano())
	chatIDStr := strconv.FormatUint(uint64(update.Message.Chat.ID), 10)
	messages, err := db.Get(chatIDStr)
	handleRecallErr(err, update)

	if messages == nil {
		return
	}

	lang, err := db.Get(fmt.Sprintf("%s-lang", strconv.FormatUint(uint64(update.Message.Chat.ID), 10)))
	if lang == nil {
		lang = "en"
	}
	if err != nil {
		Error.Printf("Can't get chat reply language\n\tChatId: %d", update.Message.Chat.ID)
	}
	data, err := ioutil.ReadFile(fmt.Sprintf("%s/%s", b.opts.replyPath, lang.(string)))
	if err != nil {
		Error.Printf("Can't open file with replies\n\tPath: %s\n\tError: %s", b.opts.replyPath, err)
	}
	replies := strings.Split(string(data), "\n")
	replyIdx := rand.Intn(len(replies))
	msg := tgbotapi.NewMessage(update.Message.Chat.ID, replies[replyIdx])
	_, err = botAPI.Send(msg)
	if err != nil {
		Error.Printf("Can't send message\n\tChatId: %d\n\t%s", update.Message.Chat.ID, err)
	}

	fwdMessageIdx := rand.Intn(len(messages.([]int)))
	fwdMsg := tgbotapi.NewForward(update.Message.Chat.ID,
		update.Message.Chat.ID, messages.([]int)[fwdMessageIdx])
	_, err = botAPI.Send(fwdMsg)
	if err != nil {
		Error.Printf("Can't forward message\n\tChatId: %d\n\t%s", update.Message.Chat.ID, err)
	}
}

func (b bot) language(db DB, update tgbotapi.Update, lang ...string) {
	idStr := strconv.FormatUint(uint64(update.Message.Chat.ID), 10)
	language := update.Message.Text
	if lang != nil {
		language = lang[0]
	}
	err := db.Put(fmt.Sprintf("%s-lang", idStr), language)
	if err != nil {
		Error.Printf("Can't set language\n\tChatId: %d\n\tLanguage: %s\n\t%s",
			update.Message.Chat.ID, update.Message.Text, err)
	}
}

func (b bot) watcher(db DB, ch chan tgbotapi.Update, botAPI *tgbotapi.BotAPI) {
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
					b.recall(db, update, botAPI)
				}
				lastUpdateDate = now
			}
		}
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
	db := NewDB(b.opts.dbPath, o)
	defer db.Close()

	botAPI, err := tgbotapi.NewBotAPI(b.token)
	if err != nil {
		Error.Println("Can't authenticate with given token")
		panic(err)
	}

	Info.Printf("Authorized on account %s", botAPI.Self.UserName)

	u := tgbotapi.NewUpdate(0)
	u.Timeout = 60

	updates, err := botAPI.GetUpdatesChan(u)
	ch := make(chan tgbotapi.Update, 1)
	go b.watcher(db, ch, botAPI)

	for update := range updates {
		if update.Message == nil {
			continue
		}

		Info.Printf("[%s] %s", update.Message.From.UserName, update.Message.Text)

		switch update.Message.Command() {
		case "start":
			go welcome(update, botAPI)
			go b.language(db, update, "en")
		case "help":
			go welcome(update, botAPI)
		case "recall":
			go b.recall(db, update, botAPI)
		case "ru", "en":
			go b.language(db, update)
		}

		b.remember(db, update)
		ch <- update
	}
}
