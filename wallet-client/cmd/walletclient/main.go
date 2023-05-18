package main

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"

	"github.com/celsopires1999/fc-ms-wallet-client/internal/database"
	replicateaccount "github.com/celsopires1999/fc-ms-wallet-client/internal/usecase/replicate_account"
	"github.com/celsopires1999/fc-ms-wallet-client/pkg/kafka"
	"github.com/celsopires1999/fc-ms-wallet-client/pkg/uow"
	ckafka "github.com/confluentinc/confluent-kafka-go/kafka"
	_ "github.com/go-sql-driver/mysql"
)

func main() {
	fmt.Println("*** GoAppClient Started ***")
	db, err := sql.Open("mysql", fmt.Sprintf("%s:%s@tcp(%s:%s)/%s?charset=utf8&parseTime=True&loc=Local", "root", "root", "mysql-client", "3306", "wallet"))
	if err != nil {
		panic(err)
	}
	defer db.Close()

	ctx := context.Background()
	uow := uow.NewUow(ctx, db)
	uow.Register("AccountDB", func(tx *sql.Tx) interface{} {
		return database.NewAccountDB(db)
	})
	uc := replicateaccount.NewReplicateAccountUseCase(uow)

	var msgChan = make(chan *ckafka.Message)

	configMapConsumer := &ckafka.ConfigMap{
		"bootstrap.servers": "kafka:29092",
		"client.id":         "goclient-consumer",
		"group.id":          "wallet",
		"auto.offset.reset": "earliest",
	}
	topics := []string{"balances"}
	consumer := kafka.NewConsumer(configMapConsumer, topics)

	go consumer.Consume(msgChan)

	for msg := range msgChan {
		var input replicateaccount.Message
		fmt.Println(string(msg.Value))
		json.Unmarshal(msg.Value, &input)
		err := uc.Execute(ctx, input.Payload)
		if err != nil {
			fmt.Println(err)
		}
	}
	fmt.Println("*** GoAppClient Finished ***")
}
