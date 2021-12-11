package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"os"
	"os/exec"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/AbnormalReality/GB_best_go1/kurs/app1/config"
	"github.com/AbnormalReality/GB_best_go1/kurs/app1/filter"
	"github.com/AbnormalReality/GB_best_go1/kurs/app1/iter"
	"github.com/AbnormalReality/GB_best_go1/kurs/app1/logger"
	"github.com/sirupsen/logrus"
)

var (
	//флаги
	debug    = flag.Bool("debug", false, "debug info")
	loglevel = flag.Int("lev", 1, "level of logging 3-Debug, 2-Warning, 1-Info")
	flCols   = flag.String("cols", "", "columns list, separated by comma(not given "+
		"= all columns, -cols=\"?\" - list)")
)

func main() {
	
	basepath, err := os.Getwd()
	if err != nil {
		log.Fatalf("path error %v ", err)
	}

	cmd := exec.Command("git", "log", "--pretty=format:\"%h - %an, %ar : %s\"")
	stdout, err := cmd.Output()
	if err != nil {
		log.Printf("Command finished with error: %v", err)
	}
	strout := string(stdout)

	strouts := strings.Split(strout, "\n")

	//fmt.Println(strouts[0])
	fmt.Printf("base path= %s git last commit version=%s \n", basepath, strouts[0])

	/*
		args := `third > 0.4 AND first = "www" OR second = "Kesha"`
		args = `third = 4.2 XOR first != "www"`
	*/
	args := `crimedescr = "459 PC  BURGLARY VEHICLE"`
	pathfile := basepath[:len(basepath)-3] + "/test1.csv"

	fmt.Println(pathfile)
	// init flag / loggers
	flag.Parse()

	//access.log error.log init
	accesslogger, err := logger.InitLogger("access.log", *debug, *loglevel)
	if err != nil {
		log.Fatalf("cannot init access.log. exit ")
	}

	errorlogger, err := logger.InitLogger("error.log", *debug, *loglevel)
	if err != nil {
		log.Fatalf("cannot init error.log. exit ")
	}

	accesslogger.Info("1 Starting the application...")

	errorlogger.Infof("1 Starting the application...")

	// load config
	c, err := config.New(basepath[:len(basepath)-3] + "/config/.env")
	if err != nil {
		errorlogger.Errorf("config error : %v", err)
		return
	}
	accesslogger.Infof("config struct : %+v", c)
	accesslogger.Infof("program flags : -cols : %s", *flCols)
	accesslogger.Infof("user query : %s", args)
	accesslogger.Infof("user file : %s", pathfile)
	accesslogger.Infof("user press ctr+c to stop any time.. :")

	_, err = pingAnalize(pathfile, args, *flCols, c.Timeout, errorlogger)
	if err != nil {
		errorlogger.Errorf("AnalizeError: %v", err)
	}
	/*
		for _, _ = range resulttext {
			//fmt.Println(str)
		}
	*/
	errorlogger.Infof("Finished the application..")
	accesslogger.Infof("Finished the application..")
}

// ловим сигналы выключения
func catchOs(cancel context.CancelFunc, errorlogger *logrus.Logger) {
	osSignalChan := make(chan os.Signal)

	signal.Notify(osSignalChan,
		syscall.SIGINT,
		syscall.SIGTERM,
	)

	select {
	case sig := <-osSignalChan:
		errorlogger.Errorf("got %s signal", sig.String())
		switch sig.String() {
		case "interrupt":
			cancel()
			return
		case "quit":
			cancel()
			return

		}
	}

}

//main func which makes the job reads and analizes csv
func readAnalizeFile(pathfile string, args string, flcols string, errorlogger *logrus.Logger) ([]string, error) {

	//check delimeters in file
	delims := iter.CheckDelimiters(pathfile)

	lineDel, err := iter.CheckEndLineDelimiters(pathfile)
	if err != nil {
		return nil, err
	}
	//read file in a python generator fashion
	// fmt.Println(delims)
	reader, err := iter.ReadLinesReadString(pathfile, lineDel)
	if err != nil {
		return nil, err
	}

	filerowidx := 0

	var filter *filter.Filter

	var textout []string

	for line := range reader {
		time.Sleep(5 * time.Millisecond)
		//fmt.Println(line)
		//line from file remove \n or \r if we have it
		line = strings.TrimRight(line, "\r\n")
		if filerowidx == 0 {
			// first row get columns names in file
			fileCols := strings.Split(line, delims[0])
			// process get columns name from cmd
			flgCols := strings.Split(flcols, ",")
			// check if we need to print cols info
			if flcols == "?" {
				fmt.Printf("columns: %s\n", line)
				break
			}
			// we get map colsMask of keys columns, and matched cols will have 1 s values
			// colsIdx has key of columns and val index in row
			// add cols to struct init filter
			if flcols != "" {
				err = validateCols(fileCols, flgCols)
				if err != nil {
					//errorlogger.Errorf("error validating columns: %v", err)
					return nil, err
				}
			}

			filter, err = filter.New(args, fileCols, flgCols)
			if err != nil {
				errorlogger.Errorf("error creating filter: ( filerow=%d ) %v", filerowidx, err)
				return nil, err
			}
			filerowidx++
			continue
		}
		// line of data here
		filerow := strings.Split(line, delims[0])
		// execute method filter if condition is ok then 'out' will be 'filerow' (only masked values)
		out, err := filter.Filter(filerow)
		if err != nil {
			//errorlogger.Errorf("message when filter processing: (filerow=%d) %v", filerowidx, err)
		}
		// print results online (and) store it
		if out != "" {
			fmt.Printf("result: %s\n", out)
			textout = append(textout, out)
		}
		filerowidx++
	}

	return textout, nil
}

// Result результат понг функции ввиде структуры
type Result struct {
	ResultError error
	TextOut     []string
}

// Пинг функция для readAnalize
// на входе парамсы, timeout в сек.
// на выходе строки результата и ошибки
func pingAnalize(pathfile string, args string, flcols string, timeout int, errorlogger *logrus.Logger) ([]string, error) {
	ctx, cancelFunc := context.WithTimeout(context.Background(), time.Duration(timeout)*time.Second)
	// т.е через таймаут будет завершена по любому эта функция ping
	defer cancelFunc()
	// check for OS signal ctrl+c
	go catchOs(cancelFunc, errorlogger)
	// канал ошибок и данных
	doneCh := make(chan Result)
	go func(ctx context.Context) {
		//pong это функция проверки сервиса возвращает ответ ошибки и результат,
		//которые пишутся в канал doneCh
		textout, err := readAnalizeFile(pathfile, args, flcols, errorlogger)
		resultRead := Result{ResultError: err, TextOut: textout}
		doneCh <- resultRead

	}(ctx)

	var res Result
	// селект блокирует ф которая завершится по ctx.Done (которая приходит по cancel)
	// второй случай pong вернула какое то значение ошибки или нет ошибок
	// ctx.Err это ошибка контекста - сработал таймаут
	select {
	case <-ctx.Done():
		res.ResultError = ctx.Err()
	case res = <-doneCh:
	}
	return res.TextOut, res.ResultError
}

//validate file cols and flgcols if any specified
func validateCols(fileCols []string, flgCols []string) error {
	var er []string

	for _, flgcls := range flgCols {
		notfound := true
		for _, filecls := range fileCols {
			if flgcls == filecls {
				notfound = false
				break
			}
		}
		if notfound {
			errstr := fmt.Sprintf("column %s not found in file columns", flgcls)
			er = append(er, errstr)
		}
	}
	if len(er) > 0 {
		bigerrstr := strings.Join(er, ", ")
		err := fmt.Errorf("errors when validating -columns flag: %s (%s)", bigerrstr, fileCols)
		return err
	}
	return nil
}