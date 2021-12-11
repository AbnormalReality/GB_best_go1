package iter

import (
	"bufio"
	"fmt"
	"io"
	"log"
	"os"
	"strings"

	"github.com/csimplestring/go-csv/detector"
)

// ReadlinesScanner is an iterator that returns one line of a file at a time.
func ReadlinesScanner(path string) (<-chan string, error) {
	file, err := os.Open(path)

	if err != nil {
		return nil, err
	}

	scanner := bufio.NewScanner(file)
	if err := scanner.Err(); err != nil {
		return nil, err
	}
	// prepare buffer for the scanner (per one line)
	buf := make([]byte, 0, 64*1024) //64k
	scanner.Buffer(buf, 1024*1024)  //1M max

	chnl := make(chan string)
	go func() {
		// Не забываем закрыть файл при выходе из функции
		defer func() {
			err := file.Close()
			if err != nil {
				log.Printf("cant close file: %v", err)
			}
		}()

		for scanner.Scan() {
			chnl <- scanner.Text()
		}
		if err := scanner.Err(); err != nil {
			fmt.Fprintf(os.Stderr, "error scanning a string: %v\n", err)
		}
		close(chnl)
	}()

	return chnl, nil
}

// CheckDelimiters Проверка на то какой разделитель будет в csv файле ;,
func CheckDelimiters(path string) []string {
	detector := detector.New()

	file, err := os.OpenFile(path, os.O_RDONLY, os.ModePerm)
	if err != nil {
		os.Exit(1)
	}
	defer file.Close()

	delimiters := detector.DetectDelimiter(file, '"')
	fmt.Println(delimiters)
	return delimiters
}

// CheckEndLineDelimiters собственный поиск конца строки в файле (умный)
func CheckEndLineDelimiters(path string) (byte, error) {
	file, err := os.OpenFile(path, os.O_RDONLY, os.ModePerm)
	if err != nil {
		os.Exit(1)
	}
	defer file.Close()

	buf := make([]byte, 400)
	if _, err := io.ReadFull(file, buf); err != nil {
		//log.Printf("error opening file %v", err)
	}

	out := string(buf)

	//fmt.Println(out)

	// -1 means no such symbol
	delidxr := strings.Index(out, "\r")
	delidxn := strings.Index(out, "\n")

	if delidxr == -1 && delidxn == 1 {
		// coudnot find delimeter error not \n nor \r
		err := fmt.Errorf("coudn't find \r \n in file this is not good")
		return 0, err
	}

	if delidxn != -1 && delidxr == -1 {
		//only r
		return '\n', nil
	}

	if delidxn == -1 && delidxr != -1 {
		//only r
		return '\r', nil
	}

	//both \n\r
	return '\n', nil

}

// ReadLinesReadString чтение строки файла путем readstring (можно указать признак конца строки)
// считанная строка отправляется в канал
func ReadLinesReadString(fn string, delim byte) (<-chan string, error) {
	//fmt.Println("readFileWithReadString")

	file, err := os.Open(fn)
	if err != nil {
		return nil, err
	}

	// Start reading from the file with a reader.
	reader := bufio.NewReader(file)

	chnl := make(chan string)
	go func() {
		// Не забываем закрыть файл при выходе из функции
		defer func() {
			var ferr = file.Close()
			if ferr != nil {
				log.Printf("cant close file: %v", ferr)
			}
		}()

		var line string
		for {

			line, err = reader.ReadString(delim)
			if err != nil || err == io.EOF {
				break
			}
			//fmt.Printf(" > Read %d characters\n line=%s", len(line), line)
			chnl <- line
		}

		if err != io.EOF {
			fmt.Printf(" > Failed with error: %v\n", err)
		}
		close(chnl)
	}()

	return chnl, nil
}