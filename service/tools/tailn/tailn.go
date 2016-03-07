package main

import (
	"fmt"
	"io"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/bitly/go-nsq"
	"github.com/codegangsta/cli"
	. "github.com/gonet2/libs/nsq-logger"
	"github.com/pquerna/ffjson/ffjson"
)

const (
	LAYOUT = "2006/1/2 15:04:05"
)

var (
	logTemplate = make(map[byte]string)
)

func init() {
	logTemplate[FINEST] = "\033[1;34m%v [FINEST] %v %v %v %v %v \033[0m"
	logTemplate[FINE] = "\033[0;34m%v [FINE] %v %v %v %v %v \033[0m"
	logTemplate[DEBUG] = "\033[1;32m%v [DEBUG] %v %v %v %v %v \033[0m"
	logTemplate[TRACE] = "\033[0;37m%v [TRACE] %v %v %v %v %v \033[0m"
	logTemplate[WARNING] = "\033[0;33m%v [WARNING] %v %v %v %v %v \033[0m"
	logTemplate[INFO] = "\033[0;32m%v [INFO] %v %v %v %v %v  \033[0m"
	logTemplate[ERROR] = "\033[0;31m%v [ERROR] %v %v %v %v %v \033[0m"
	logTemplate[CRITICAL] = "\033[7;31m%v [CRITICAL] %v %v %v %v %v \033[0m"
}

var (
	tailFlag = []cli.Flag{
		cli.StringFlag{
			Name:  "topic,t",
			Value: "LOG",
			Usage: "NSQ topic , defalut is LOG",
		},
		cli.StringFlag{
			Name:  "channel, c",
			Value: "tailn",
			Usage: "NSQ channel, defalut is tailn",
		},
		cli.StringFlag{
			Name:  "number, n",
			Value: "0",
			Usage: "Line to show, defalut no limit",
		},
		cli.StringFlag{
			Name:  "nsqd-tcp-address, a",
			Value: "localhost:4150",
			Usage: "nsqd TCP address",
		},
		cli.StringFlag{
			Name:  "lookupd-http-address, l",
			Value: "localhost:4161",
			Usage: "lookupd HTTP address",
		},
		cli.StringFlag{
			Name:  "timeout, o",
			Value: "5",
			Usage: "Dial timeout, default 5s",
		},
		cli.StringFlag{
			Name:  "type",
			Value: "NSQLOG",
			Usage: "Tail type , default NSQLOG (others print json )",
		},
		cli.StringFlag{
			Name:  "log",
			Value: "false",
			Usage: "whether open inner log",
		},
		cli.StringFlag{
			Name:  "tofile, f",
			Value: "",
			Usage: "output to file, defualt is write into current dir",
		},
	}
)

type TailHandler struct {
	totalMessages int
	messagesShown int
	writer        io.Writer
	printMessage  func(io.Writer, *nsq.Message) error
}

func NSQLog(w io.Writer, m *nsq.Message) error {
	info := &LogFormat{}
	err := ffjson.Unmarshal(m.Body, &info)
	if err != nil {
		fmt.Printf("err %v\n", err)
		return nil
	}
	_, err = fmt.Fprintln(w, fmt.Sprintf(logTemplate[info.Level], info.Time.Format(LAYOUT), info.Prefix, info.Host, info.Msg, info.Caller, info.LineNo))
	if err != nil {
		return err
	}
	return nil
}

func Log(w io.Writer, m *nsq.Message) error {
	_, err := fmt.Fprintln(w, m.Body)
	if err != nil {
		return err
	}
	return nil
}

func LogFile(w io.Writer, m *nsq.Message) error {
	info := &LogFormat{}
	err := ffjson.Unmarshal(m.Body, &info)
	if err != nil {
		fmt.Printf("err %v\n", err)
		return nil
	}
	line := fmt.Sprintf("%v [%v] %v %v %v %v %v", info.Time.Format(LAYOUT), info.Level, info.Prefix, info.Host, info.Msg, info.Caller, info.LineNo)
	_, err = fmt.Fprintln(w, line)
	if err != nil {
		return err
	}
	return nil

}

func main() {
	myApp := cli.NewApp()
	myApp.Name = "tailn"
	myApp.Usage = "Tail log from nsq !"
	myApp.Version = "0.0.2"
	myApp.Flags = tailFlag
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)
	myApp.Action = func(c *cli.Context) {
		// TODO CheckFlag
		CheckFlag(c)
		cfg := nsq.NewConfig()
		// dail timeout
		cfg.DialTimeout = time.Duration(c.Int("timeout")) * time.Second
		cfg.UserAgent = fmt.Sprintf("go-nsq version:%v", nsq.VERSION)
		cfg.MaxInFlight = 128
		consumer, err := nsq.NewConsumer(c.String("topic"), c.String("channel"), cfg)
		if err != nil {
			fmt.Printf("error %v\n", err)
			os.Exit(0)
		}

		f := NSQLog
		var w io.Writer = os.Stdout
		if c.String("type") != "NSQLOG" {
			f = Log
		}

		if c.String("tofile") != "" {
			f = LogFile
			fl, err := os.OpenFile(c.String("tofile"), os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
			if err != nil {
				fmt.Printf("error %v\n", err)
				os.Exit(0)
			}
			w = fl
		}

		// 屏蔽内部日志
		if !c.Bool("log") {
			consumer.SetLogger(nil, 0)
		}

		consumer.AddHandler(&TailHandler{c.Int("number"), 0, w, f})
		err = consumer.ConnectToNSQDs(strings.Split(c.String("nsqd-tcp-address"), ","))
		if err != nil {
			fmt.Printf("error %v\n", err)
			os.Exit(0)

		}
		err = consumer.ConnectToNSQLookupds(strings.Split(c.String("lookupd-http-address"), ","))
		if err != nil {
			fmt.Printf("error %v\n", err)
			os.Exit(0)
		}
		for {
			select {
			case <-consumer.StopChan:
				return
			case <-sigChan:
				consumer.Stop()
			}
		}
	}
	myApp.Run(os.Args)
}

func CheckFlag(c *cli.Context) {
	if c.String("channel") == "" || c.String("topic") == "" {
		cli.ShowAppHelp(c)
		os.Exit(0)
	}
}

func (th *TailHandler) HandleMessage(m *nsq.Message) error {
	th.messagesShown++
	if err := th.printMessage(th.writer, m); err != nil {
		fmt.Printf("err %v\n", err)
		return err
	}
	if th.totalMessages > 0 && th.totalMessages < th.messagesShown {
		os.Exit(0)
	}
	return nil
}
