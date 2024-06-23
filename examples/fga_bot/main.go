package main

import (
	"context"
	"os"
	"os/signal"
	"log"
	"strings"
	"errors"
	"math/rand"
	"time"

	"github.com/go-telegram/bot"
	"github.com/go-telegram/bot/models"

	"github.com/jmoiron/sqlx"
	_ "github.com/lib/pq"
	_ "database/sql"
	"database/sql/driver"
)


type Advice struct {
	Id   int        `db:"id"`
	Text string     `db:"text"`
	Tags StringList `db:"tags"`
}

type StringList []string

var db *sqlx.DB
func connectDB() {  
	DB, err := sqlx.Connect("postgres", "user=postgres dbname=fgaBot sslmode=disable")
	if err != nil {
		log.Fatalln(err)
	}
	db = DB
}

var count int
func getCount() {
	err := db.Get(&count, "SELECT count(*) FROM advices")
	if err != nil {
		log.Fatalln(err)
	}
}

var tags = make(map[string]string)
func getTags() {
	rows, err := db.Queryx("SELECT alias, title FROM advices_tags")
	if err != nil {
		log.Fatalln(err)
	}
	for rows.Next() {
		var alias, title string
		err := rows.Scan(&alias, &title)
		if err != nil {
			log.Fatalln(err)
		}
		tags[alias] = title
	}
}

func (s *StringList) Scan(src interface{}) error {
	switch src := src.(type) {
	case []byte:
		str := string(src)
		arr := strings.Split(str[1:len(str)-1], ",")
		*s = StringList(arr)
		return nil
	case string:
		arr := strings.Split(src[1:len(src)-1], ",")
		*s = StringList(arr)
		return nil
	default:
		return errors.New("Unsupported scan type for StringList")
	}
}

func (s StringList) Value() (driver.Value, error) {
	if len(s) == 0 {
		return "{}", nil
	}
	return strings.Join(s, ","), nil
}

func preprocess() {
	log.Printf("Connecting to DB...")
	connectDB()
	log.Printf("Successful")
	getCount()
	log.Printf("Total count of advices: %d", count)
	getTags()
	log.Printf("Tags received")
}

func selectdb(db *sqlx.DB, num int) string {
	advice := Advice{}
	rows, err := db.Queryx("SELECT * FROM advices where id=$1", num)
	if err != nil {
		log.Fatalln(err)
	}
	for rows.Next() {
		err := rows.StructScan(&advice)
		if err != nil {
			log.Fatalln(err)
		}
	}
	return advice.Text
}

func main() {
	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt)
	defer cancel()

	preprocess()

	opts := []bot.Option{
		// bot.WithMiddlewares(getRandomAdvice),
		bot.WithDefaultHandler(defaultHandler),
	}

	b, err := bot.New(os.Getenv("EXAMPLE_TELEGRAM_BOT_TOKEN"), opts...)
	if nil != err {
		panic(err)
	}

	b.RegisterHandler(bot.HandlerTypeMessageText, "/advice", bot.MatchTypeExact, randomAdviceHandler)
	b.Start(ctx)
}

func defaultHandler(ctx context.Context, b *bot.Bot, update *models.Update) {
	log.Printf("default handler")
}

func randomAdviceHandler(ctx context.Context, b *bot.Bot, update *models.Update) {
	rand.Seed(time.Now().Unix())
	randNum := rand.Intn(count)
	advice := selectdb(db, randNum)
	b.SendMessage(ctx, &bot.SendMessageParams{
		ChatID:				update.Message.Chat.ID,
		Text:		 		advice,
    	ReplyParameters:	&models.ReplyParameters{
        	MessageID: update.Message.ID,
    		},
	})
	if update.Message != nil {
		log.Printf("for %s bot say: %s", update.Message.From.FirstName, advice)
	}
}
